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

	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

func getFile() *excelize.File {
	excelPath := getFilePath("Planilha.xlsx")
	f, err := excelize.OpenFile(excelPath)
	if err != nil {
		panic(err)
	}
	return f
}

func getFilePath(fileName string) string {
	exePath, err := os.Executable()
	if err != nil {
		panic(err)
	}
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
var timeInfinity, _ = time.ParseInLocation("2006", "2099", location)

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

	// sendMessage := func(number, message string) error {
	// 	// whatsapp := getWhatsapp()
	// 	// _, err := whatsapp.SendMessage(context.Background(), types.NewJID(numberBeautify(number), types.DefaultUserServer), &waE2E.Message{
	// 	// 	Conversation: proto.String(message),
	// 	// })
	// 	// time.Sleep(2 * time.Second)
	// 	println(number, message)
	// 	return nil
	// }

	// headers, datas := panic("asd")

	// for i, row := range datas {
	// 	enviarEm := row["enviar em"].(time.Time).Add(-1 * time.Minute)
	// 	enviadoEm := row["enviado em"].(sql.NullTime)
	// 	fmt.Printf("enviarEm: %#v\n", enviarEm)
	// 	fmt.Printf("enviadoEm: %#v\n", enviadoEm)
	// 	shouldSend := func() bool {
	// 		if !enviadoEm.Valid && enviadoEm.Time.After(enviarEm) {
	// 			return false
	// 		} else {
	// 			return enviadoEm.Time.Before(enviarEm) && enviarEm.Before(now)
	// 		}
	// 	}()
	// 	if shouldSend {
	// 		columnName, err := excelize.ColumnNumberToName(headers["enviado em"] + 1)
	// 		if err != nil {
	// 			panic(err)
	// 		}
	// 		cell := fmt.Sprintf("%s%d", columnName, i+2)
	// 		err = sendMessage(row["whatsapp"].(string), row["mensagem"].(string))
	// 		cellContent := func() string {
	// 			if err != nil {
	// 				return err.Error()
	// 			} else {
	// 				return now.Format("02/01/2006 15:04")
	// 			}
	// 		}()
	// 		f.SetCellValue(f.GetSheetName(0), cell, cellContent)
	// 		if err := f.Save(); err != nil {
	// 			panic(err)
	// 		}
	// 	}
	// }
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
	container, err := sqlstore.New("sqlite3", "file:"+getFilePath("store.db")+"?_foreign_keys=on", dbLog)
	if err != nil {
		panic(err)
	}
	// If you want multiple sessions, remember their JIDs and use .GetDevice(jid) or .GetAllDevices() instead.
	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		panic(err)
	}
	clientLog := waLog.Stdout("Client", "DEBUG", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)

	if client.Store.ID == nil {
		// No ID stored, new login
		qrChan, _ := client.GetQRChannel(context.Background())
		err = client.Connect()
		if err != nil {
			panic(err)
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				time.Sleep(5 * time.Minute)
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else {
		err = client.Connect()
		if err != nil {
			panic(err)
		}
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

func getCell(col, row int) string {
	col += 1
	row += 1
	return bang(f.CalcCellValue(f.GetSheetName(0), bang(excelize.CoordinatesToCellName(col, row))))
}

func getCellType(col, row int) excelize.CellType {
	col += 1
	row += 1
	return bang(f.GetCellType(f.GetSheetName(0), bang(excelize.CoordinatesToCellName(col, row))))
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
