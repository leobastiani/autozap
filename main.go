package main

import (
	"bytes"
	"database/sql"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"fmt"
	"os"

	"github.com/xuri/excelize/v2"
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

const sheetName = "autozap"

var f *excelize.File
var header Header

func main() {
	f = getFile()

	defer func() {
		if err := f.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	parseHeaders()

	sendMessage := func(number, message string) error {
		req, err := http.NewRequest(
			"POST",
			"http://127.0.0.1:3000/api/sendText",
			bytes.NewBufferString(fmt.Sprintf(`{
				"chatId": "%s@c.us",
				"text": %q,
				"session": "default"
			}`, numberBeautify(number), message)),
		)
		if err != nil {
			return err
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Api-Key", os.Getenv("API_KEY"))
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		return nil
	}

	for row, i := range IterRowsWithHeader() {
		shouldSend := func() bool {
			enviarEm := row.enviarEm.Add(-1 * time.Minute)
			if enviarEm.After(now) {
				return false
			}
			if row.enviadoEm.Valid && row.enviadoEm.Time.After(enviarEm) {
				return false
			}
			return true
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

func bang[T any](t T, err error) T {
	bang0(err)
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
	bang0(f.SetCellValue(sheetName, bang(excelize.CoordinatesToCellName(col, row)), value))
}

func getCell(col, row int) string {
	col += 1
	row += 1
	return strings.TrimSpace(bang(f.CalcCellValue(sheetName, bang(excelize.CoordinatesToCellName(col, row)))))
}

var DateStyle = sync.OnceValue(func() int {
	return bang(f.NewStyle(&excelize.Style{CustomNumFmt: func() *string {
		exp := "[$-416]dd/mm/yyyy hh:mm;@"
		return &exp
	}()}))
})

func parseWithOptionalTime(value string) sql.NullTime {
	if strings.Contains(value, " ") {
		t, err := time.ParseInLocation("2/1/2006 15:04", value, location)
		if err != nil {
			return sql.NullTime{}
		}
		return sql.NullTime{Time: t, Valid: true}
	} else {
		t, err := time.ParseInLocation("2/1/2006", value, location)
		if err != nil {
			return sql.NullTime{}
		}
		t = t.Add(8 * time.Hour)
		return sql.NullTime{Time: t, Valid: true}
	}
}

func getCellTime(col, row int) sql.NullTime {
	f.SetCellStyle(
		sheetName,
		bang(excelize.CoordinatesToCellName(col+1, row+1)),
		bang(excelize.CoordinatesToCellName(col+1, row+1)),
		DateStyle(),
	)
	value := getCell(col, row)
	if value == "" {
		return sql.NullTime{}
	}
	return parseWithOptionalTime(value)
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

func GetIsRowEmpty(row int) bool {
	for i := 0; i < 10; i++ {
		if getCell(i, row) != "" {
			return false
		}
	}
	return true
}

func IterRowsWithHeader() func(func(Row, int) bool) {
	return func(yield func(Row, int) bool) {
		for i := range IterRows() {
			var row Row
			row.whatsapp = numberBeautify(getCell(header.whatsapp, i))
			if row.whatsapp == "" {
				continue
			}
			row.enviadoEm = getCellTime(header.enviadoEm, i)
			if !row.enviadoEm.Valid {
				if getCell(header.enviadoEm, i) != "" {
					continue
				}
			}
			enviarEm := getCellTime(header.enviarEm, i)
			if !enviarEm.Valid {
				continue
			}
			row.enviarEm = enviarEm.Time
			row.mensagem = getCell(header.mensagem, i)
			if row.mensagem == "" {
				continue
			}
			if !yield(row, i) {
				return
			}
		}
	}
}

func IterRows() func(func(int) bool) {
	return func(yield func(int) bool) {
		emptyConsecutive := 0
		for i := 1; ; i++ {
			if GetIsRowEmpty(i) {
				emptyConsecutive++
				if emptyConsecutive == 10 {
					return
				}
				continue
			}
			emptyConsecutive = 0
			if !yield(i) {
				return
			}
		}
	}
}

func parseHeaders() {
	for value, col := range IterRow(0) {
		value = strings.ToLower(value)
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
