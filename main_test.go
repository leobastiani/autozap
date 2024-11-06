package main

import (
	"fmt"
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

	t.Run("IterRowsWithHeader", func(t *testing.T) {
		for row := range IterRowsWithHeader() {
			fmt.Printf("whatsapp: %#v\n", row.whatsapp)
			fmt.Printf("mensagem: %#v\n", row.mensagem)
			fmt.Printf("enviarEm: %#v\n", row.enviarEm.Format("02/01/2006 15:04"))
			fmt.Printf("enviadoEm: %#v\n", row.enviadoEm.Time.Format("02/01/2006 15:04"))
			fmt.Printf("\n")
		}
	})
}
