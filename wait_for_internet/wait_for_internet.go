package main

import (
	"net/http"
	"time"
)

func main() {
	for {
		_, err := http.Get("http://clients3.google.com/generate_204")
		if err == nil {
			return
		}
		time.Sleep(60 * time.Second)
	}
}
