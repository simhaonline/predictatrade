package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"pat-engine/internal/backtest"
	"pat-engine/internal/license"
	"pat-engine/internal/provider"
	"pat-engine/internal/risk"
	"pat-engine/internal/store"
)

// cmd/gateway is the live signal backend + REST API. It ingests bars (POST /bar),
// runs the strategy pipeline, persists to TimescaleDB, publishes to Valkey, writes the
// EA signal file, and serves a JSON API the frontend/control-plane consume.
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

	registerAPI(gw)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("pat-engine gateway listening on :%s  ->  %s", port, out)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// registerAPI wires the JSON REST surface the frontend/control-plane consume.
func registerAPI(gw *provider.Gateway) {
	api := "/api/v1"

	// Health / discovery
	http.HandleFunc(api+"/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]string{"status": "ok", "broker_time_offset": itoa(gw.Execution().TimezoneOffset)})
	})

	// Strategies available to this license
	http.HandleFunc(api+"/strategies", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"strategies": gw.Strategies(), "license_plan": planOf(gw.License())})
	})

	// Broker execution profile (static economics + sessions)
	http.HandleFunc(api+"/broker", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, gw.Execution())
	})

	// Risk mandate (default; control-plane overrides per user/plan)
	http.HandleFunc(api+"/risk", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, risk.DefaultRisk())
	})

	// Current session in BROKER time (timezone-correct signal generation)
	http.HandleFunc(api+"/session", func(w http.ResponseWriter, r *http.Request) {
		name, overlap := gw.Policy().Session(time.Now())
		writeJSON(w, map[string]any{"session": name, "overlap": overlap, "tz_offset": gw.Execution().TimezoneOffset})
	})

	// Recent signals (audit)
	http.HandleFunc(api+"/signals", func(w http.ResponseWriter, r *http.Request) {
		n := queryInt(r, "limit", 50)
		writeJSON(w, gw.Store().RecentSignals(context.Background(), n))
	})

	// Recent bars
	http.HandleFunc(api+"/bars", func(w http.ResponseWriter, r *http.Request) {
		n := queryInt(r, "limit", 200)
		writeJSON(w, gw.Store().RecentBars(context.Background(), n))
	})

	// Device activation (hardware fingerprint -> license binding)
	http.HandleFunc(api+"/devices/activate", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			DeviceID      string `json:"device_id"`
			LicenseKey    string `json:"license_key"`
			Fingerprint   string `json:"fingerprint"`
			Components    string `json:"components"`
			InstallationID string `json:"installation_id"`
			Hostname      string `json:"hostname"`
			OS            string `json:"os"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		gw.Store().UpsertDevice(context.Background(), store.Device{
			ID: body.DeviceID, LicenseID: body.LicenseKey, Fingerprint: body.Fingerprint,
			Components: body.Components, InstallID: body.InstallationID, Hostname: body.Hostname, OS: body.OS,
		})
		writeJSON(w, map[string]string{"status": "activated", "device_id": body.DeviceID})
	})

	// Agent heartbeat / telemetry
	http.HandleFunc(api+"/devices/heartbeat", func(w http.ResponseWriter, r *http.Request) {
		var t store.Telemetry
		if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		gw.Store().SaveTelemetry(context.Background(), t)
		writeJSON(w, map[string]string{"status": "ok"})
	})

	// License validation (signed token + optional device binding)
	http.HandleFunc(api+"/licensing/validate", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			LicenseKey string `json:"license_key"`
			DeviceID   string `json:"device_id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		secret := os.Getenv("PAT_LICENSE_SECRET")
		if secret == "" {
			secret = license.DefaultDevSecret
		}
		lic, err := license.Parse(body.LicenseKey, secret)
		if err != nil {
			writeJSON(w, map[string]string{"status": "invalid", "reason": err.Error()})
			return
		}
		if err := lic.IsValid(body.DeviceID); err != nil {
			writeJSON(w, map[string]string{"status": "invalid", "reason": err.Error()})
			return
		}
		// Store-backed device binding: if the token is bound to a device, that device
		// must be registered (the agent's hardware fingerprint was seen). Without the
		// store we cannot verify, so we only enforce when persistence is available.
		if lic.DeviceID != "" && gw.Store() != nil {
			dev := gw.Store().GetDevice(r.Context(), lic.DeviceID)
			if dev == nil {
				writeJSON(w, map[string]string{
					"status": "invalid",
					"reason": "device_not_registered:" + lic.DeviceID,
				})
				return
			}
		}
		writeJSON(w, map[string]any{
			"status":             "active",
			"plan":               lic.Plan,
			"allowed_strategies": lic.AllowedStrategies,
			"device_binding":     lic.DeviceID != "",
		})
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func queryInt(r *http.Request, key string, def int) int {
	if v := r.URL.Query().Get(key); v != "" {
		if n, err := atoi(v); err == nil {
			return n
		}
	}
	return def
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var b [12]byte
	pos := len(b)
	for i > 0 {
		pos--
		b[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		b[pos] = '-'
	}
	return string(b[pos:])
}

func atoi(s string) (int, error) {
	n := 0
	neg := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if i == 0 && c == '-' {
			neg = true
			continue
		}
		if c < '0' || c > '9' {
			return 0, errBadInt
		}
		n = n*10 + int(c-'0')
	}
	if neg {
		n = -n
	}
	return n, nil
}

var errBadInt = &numError{}

type numError struct{}

func (e *numError) Error() string { return "invalid number" }

func planOf(l *license.License) string {
	if l == nil {
		return "DEV"
	}
	return l.Plan
}
