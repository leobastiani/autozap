package main

import (
	"database/sql"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"context"
	"fmt"
	"os"

	"github.com/xuri/excelize/v2"
	"google.golang.org/protobuf/proto"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
)

func getFile() *excelize.File {
	excelPath := getFilePath("Planilha.xlsx")
	f := bang(excelize.OpenFile(excelPath))
	return f
}

func getFilePath(fileName string) string {
	exePath := bang(os.Executable())
	return filepath.Join(filepath.Dir(exePath), fileName)
}

type Header struct {
	whatsapp  int
	enviarEm  int
	enviadoEm int
	mensagem  int
}

type Row struct {
	whatsapp  string
	enviarEm  time.Time
	enviadoEm sql.NullTime
	mensagem  string
}

var now = time.Now()
var location = now.Location()

var f *excelize.File
var header Header

func main() {
	defer cleanupWhatsapp()
	f = getFile()

	defer func() {
		if err := f.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	parseHeaders()

	sendMessage := func(number, message string) error {
		whatsapp := getWhatsapp()
		_, err := whatsapp.SendMessage(context.Background(), types.NewJID(numberBeautify(number), types.DefaultUserServer), &waE2E.Message{
			Conversation: proto.String(message),
		})
		time.Sleep(2 * time.Second)
		return err
		// fmt.Println("number: %#v, message: %#v", number, message)
		// return nil
	}

	for row, i := range IterRowsWithHeader() {
		shouldSend := func() bool {
			if row.enviadoEm.Valid && row.enviadoEm.Time.After(row.enviarEm) {
				return false
			} else {
				return row.enviarEm.Before(now)
			}
		}()
		if shouldSend {
			err := sendMessage(row.whatsapp, row.mensagem)
			cellContent := func() string {
				if err != nil {
					return err.Error()
				} else {
					return now.Format("02/01/2006 15:04")
				}
			}()
			setCell(header.enviadoEm, i, cellContent)
			bang0(f.Save())
		}
	}
}

func numberBeautify(number string) string {
	// remove non numeric character from number
	number = regexp.MustCompile(`[^0-9]`).ReplaceAllString(number, "")
	if len(number) == 11 {
		return "55" + number
	}
	return number
}

func createWhatsapp() *whatsmeow.Client {
	dbLog := waLog.Stdout("Database", "DEBUG", true)
	container := bang(sqlstore.New("sqlite3", "file:"+getFilePath("store.db")+"?_foreign_keys=on", dbLog))
	// If you want multiple sessions, remember their JIDs and use .GetDevice(jid) or .GetAllDevices() instead.
	deviceStore := bang(container.GetFirstDevice())
	clientLog := waLog.Stdout("Client", "DEBUG", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)

	if client.Store.ID == nil {
		// No ID stored, new login
		qrChan, _ := client.GetQRChannel(context.Background())
		bang0(client.Connect())
		for evt := range qrChan {
			if evt.Event == "code" {
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				time.Sleep(5 * time.Minute)
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else {
		bang0(client.Connect())
	}
	cleanupWhatsapp = func() {
		client.Disconnect()
	}
	return client
}

var getWhatsapp = sync.OnceValue(createWhatsapp)
var cleanupWhatsapp = func() {}

func bang[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}
	return t
}

func bang0(err error) {
	if err != nil {
		panic(err)
	}
}

func setCell(col, row int, value any) {
	col += 1
	row += 1
	bang0(f.SetCellValue(f.GetSheetName(0), bang(excelize.CoordinatesToCellName(col, row)), value))
}

func getCell(col, row int) string {
	col += 1
	row += 1
	return bang(f.CalcCellValue(f.GetSheetName(0), bang(excelize.CoordinatesToCellName(col, row))))
}

func getCellTime(col, row int) sql.NullTime {
	value := getCell(col, row)
	if value == "" {
		return sql.NullTime{}
	}
	t, err := time.ParseInLocation("2/1/2006 15:04", value, location)
	if err != nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: t, Valid: true}
}

func IterRow(row int) func(func(string, int) bool) {
	return func(yield func(string, int) bool) {
		for i := 0; ; i++ {
			value := getCell(i, row)
			if value == "" {
				return
			}
			if !yield(value, i) {
				return
			}
		}
	}
}

func IterRowsWithHeader() func(func(Row, int) bool) {
	return func(yield func(Row, int) bool) {
		for i := 1; ; i++ {
			var row Row
			row.whatsapp = numberBeautify(getCell(header.whatsapp, i))
			if row.whatsapp == "" {
				return
			}
			row.enviadoEm = getCellTime(header.enviadoEm, i)
			if !row.enviadoEm.Valid {
				if getCell(header.enviadoEm, i) != "" {
					continue
				}
			}
			row.enviarEm = getCellTime(header.enviarEm, i).Time.Add(-1 * time.Minute)
			row.mensagem = getCell(header.mensagem, i)
			if !yield(row, i) {
				return
			}
		}
	}
}

func parseHeaders() {
	for value, col := range IterRow(0) {
		value = strings.ToLower(strings.TrimSpace(value))
		switch value {
		case "whatsapp":
			header.whatsapp = col
		case "enviar em":
			header.enviarEm = col
		case "enviado em":
			header.enviadoEm = col
		case "mensagem":
			header.mensagem = col
		}
	}
}
