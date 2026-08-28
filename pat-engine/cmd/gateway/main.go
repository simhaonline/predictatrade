package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"pat-engine/internal/backtest"
	"pat-engine/internal/license"
	"pat-engine/internal/provider"
	"pat-engine/internal/store"
)

// cmd/gateway is the live signal backend. It ingests bars from a Windows Agent
// (POST /bar), runs the strategy pipeline, persists bars/signals to TimescaleDB,
// publishes live signals via Valkey, and writes the EA signal file. If the datastore
// is unreachable it degrades to in-memory and keeps serving signals.
func main() {
	out := os.Getenv("SIGNAL_FILE")
	if out == "" {
		out = "signals/PAT_signals.txt"
	}
	gw := provider.New(nil, out)

	// Persistence (TimescaleDB + Valkey). Degrades to in-memory if unavailable.
	if dsn := os.Getenv("PAT_DB_DSN"); dsn != "" {
		st := store.New(context.Background(), dsn, os.Getenv("PAT_REDIS_URL"))
		gw.SetStore(st)
		pg, vk := st.Healthy()
		log.Printf("store: postgres=%v valkey=%v", pg, vk)
	} else {
		log.Println("no PAT_DB_DSN set — persistence disabled (in-memory only)")
	}

	if tok := os.Getenv("PAT_LICENSE"); tok != "" {
		secret := os.Getenv("PAT_LICENSE_SECRET")
		if secret == "" {
			secret = license.DefaultDevSecret
		}
		if err := gw.LoadLicense(tok, secret); err != nil {
			log.Fatalf("invalid PAT_LICENSE: %v", err)
		}
		log.Println("license loaded from PAT_LICENSE")
	} else {
		log.Println("no PAT_LICENSE set — using DEV license (all strategies allowed)")
	}

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
