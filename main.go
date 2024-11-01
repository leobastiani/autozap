package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
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
	f := getFile()
	if err := f.Save(); err != nil {
		panic(err)
	}

	defer func() {
		if err := f.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	sendMessage := func(number, message string) {
		fmt.Printf("number: %#v\n", number)
		fmt.Printf("message: %#v\n", message)
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
			sendMessage(row["whatsapp"].(string), row["mensagem"].(string))
			f.SetCellValue(f.GetSheetName(0), cell, now.Format("02/01/2006 15:04"))
			if err := f.Save(); err != nil {
				panic(err)
			}
		}
	}
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
								panic(err)
							}
							data[header] = sql.NullTime{
								Valid: true,
								Time:  t,
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
