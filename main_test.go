package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xuri/excelize/v2"
)

func TestLeo(t *testing.T) {
	f = bang(excelize.OpenFile("test.xlsx"))
	defer bang0(f.Close())

	t.Run("correct headers", func(t *testing.T) {
		parseHeaders()
		assert.Equal(t, header, Header{
			whatsapp:  1,
			enviarEm:  2,
			enviadoEm: 3,
			mensagem:  4,
		})
	})

	t.Run("asd", func(t *testing.T) {
		for range IterRowsWithHeader() {
			// fmt.Printf("row: %#v\n", row)
		}
	})
}
