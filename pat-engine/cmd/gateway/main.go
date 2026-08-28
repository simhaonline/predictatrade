package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"pat-engine/internal/backtest"
	"pat-engine/internal/provider"
)

// cmd/gateway is the live signal backend. It ingests bars from a Windows Agent
// (POST /bar), runs the strategy pipeline, and writes executable signals to the
// EA signal file. Zero external dependencies.
func main() {
	out := os.Getenv("SIGNAL_FILE")
	if out == "" {
		out = "signals/PAT_signals.txt"
	}
	gw := provider.New(nil, out)

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})
	http.HandleFunc("/signal", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(gw.Latest()))
	})
	http.HandleFunc("/bar", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var b backtest.Bar
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		gw.IngestBar(b)
		w.Write([]byte("OK"))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("pat-engine gateway listening on :%s  ->  %s", port, out)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
