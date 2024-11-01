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
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

func getFile() *excelize.File {
	exePath, err := os.Executable()
	if err != nil {
		panic(err)
	}
	excelPath := filepath.Join(filepath.Dir(exePath), "Planilha.xlsx")
	f, err := excelize.OpenFile(excelPath)
	if err != nil {
		panic(err)
	}
	return f
}

type Row map[string]any
type Headers map[string]int

var now = time.Now()
var location = now.Location()

func main() {
	defer cleanupWhatsapp()
	f := getFile()
	if err := f.Save(); err != nil {
		panic(err)
	}

	defer func() {
		if err := f.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	sendMessage := func(number, message string) error {
		whatsapp := getWhatsapp()
		_, err := whatsapp.SendMessage(context.Background(), types.NewJID(numberBeautify(number), types.DefaultUserServer), &waE2E.Message{
			Conversation: proto.String(message),
		})
		return err
	}

	headers, datas := getRows(f, location)

	for i, row := range datas {
		enviarEm := row["enviar em"].(time.Time).Add(-1 * time.Minute)
		enviadoEm := row["enviado em"].(sql.NullTime)
		shouldSend := func() bool {
			if !enviadoEm.Valid && enviadoEm.Time.After(enviarEm) {
				return false
			} else {
				return enviadoEm.Time.Before(enviarEm) && enviarEm.Before(now)
			}
		}()
		if shouldSend {
			columnName, err := excelize.ColumnNumberToName(headers["enviado em"] + 1)
			if err != nil {
				panic(err)
			}
			cell := fmt.Sprintf("%s%d", columnName, i+2)
			err = sendMessage(row["whatsapp"].(string), row["mensagem"].(string))
			cellContent := func() string {
				if err != nil {
					return err.Error()
				} else {
					return now.Format("02/01/2006 15:04")
				}
			}()
			f.SetCellValue(f.GetSheetName(0), cell, cellContent)
			if err := f.Save(); err != nil {
				panic(err)
			}
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

func getRows(f *excelize.File, location *time.Location) (headers Headers, datas []Row) {
	datas = []Row{}
	headers = Headers{}

	rows, err := f.Rows(f.GetSheetName(0))
	if err != nil {
		panic(err)
	}
	if rows.Next() {
		row, err := rows.Columns()
		if err != nil {
			panic(err)
		}
		for i, colCell := range row {
			smallCell := strings.ToLower(strings.TrimSpace(colCell))
			if smallCell == "enviar em" || smallCell == "enviado em" || smallCell == "whatsapp" || smallCell == "mensagem" {
				headers[smallCell] = i
			}
		}
	}

	rowN := 0
	for rows.Next() {
		rowN += 1
		row, err := rows.Columns()
		if err != nil {
			panic(err)
		}
		data := Row{}
		for i, colCell := range row {
			for header, j := range headers {
				if j == i {
					if header == "enviar em" {
						t, err := time.ParseInLocation("02/01/2006 15:04", colCell, location)
						if err != nil {
							panic(err)
						}
						data[header] = t
					} else if header == "enviado em" {
						if colCell == "" {
							data[header] = sql.NullTime{}
						} else {
							t, err := time.ParseInLocation("02/01/2006 15:04", colCell, location)
							if err != nil {
								data[header] = sql.NullTime{}
							} else {
								data[header] = sql.NullTime{
									Valid: true,
									Time:  t,
								}
							}
						}
					} else {
						data[header] = strings.TrimSpace(colCell)
					}
				}
			}
		}
		datas = append(datas, data)
	}
	if err = rows.Close(); err != nil {
		panic(err)
	}
	return
}

func createWhatsapp() *whatsmeow.Client {
	dbLog := waLog.Stdout("Database", "DEBUG", true)
	container, err := sqlstore.New("sqlite3", "file:examplestore.db?_foreign_keys=on", dbLog)
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
