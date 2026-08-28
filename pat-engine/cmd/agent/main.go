package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"pat-engine/internal/backtest"
)

// cmd/agent is a reference data feeder. On a real machine it would read ticks from
// the MT terminal; here it streams bars (synthetic or from a CSV) to the gateway.
// It demonstrates the full ingest path without needing a live terminal.
func main() {
	url := os.Getenv("GATEWAY")
	if url == "" {
		url = "http://localhost:8080/bar"
	}

	var bars []backtest.Bar
	if f := os.Getenv("BARS_CSV"); f != "" {
		b, err := backtest.FromCSV(f)
		if err != nil {
			panic(err)
		}
		bars = b
	} else {
		bars = backtest.Generate(3000, 7)
	}

	client := &http.Client{}
	sent := 0
	for i, b := range bars {
		data, _ := json.Marshal(b)
		resp, err := client.Post(url, "application/json", bytes.NewReader(data))
		if err != nil {
			fmt.Println("post err:", err)
			continue
		}
		resp.Body.Close()
		sent++
		if i%500 == 0 {
			fmt.Printf("sent %d bars\n", i)
		}
	}
	fmt.Println("agent done, sent", sent, "bars")
}
