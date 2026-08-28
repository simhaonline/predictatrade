package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"pat-engine/internal/agentlib"
	"pat-engine/internal/backtest"
	"pat-engine/internal/store"
)

// cmd/agent is a reference data feeder. On a real machine it would read ticks from
// the MT terminal; here it streams bars (synthetic or from a CSV) to the gateway.
// It also registers its hardware fingerprint and sends telemetry heartbeats so the
// backend can bind the license to the device and detect misuse.
func main() {
	url := os.Getenv("GATEWAY")
	if url == "" {
		url = "http://localhost:8080/bar"
	}

	fp := agentlib.Collect()
	api := os.Getenv("AGENT_API")
	if api != "" {
		activate(api, fp)
		go heartbeat(api, fp)
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

	client := &http.Client{Timeout: 5 * time.Second}
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

func activate(api string, fp agentlib.Fingerprint) {
	body, _ := json.Marshal(map[string]any{
		"device_id":       fp.DeviceID,
		"fingerprint":     fp.Fingerprint,
		"components":      fp.Components,
		"installation_id": fp.InstallationID,
		"hostname":        fp.Hostname,
		"os":              fp.OS,
	})
	resp, err := http.Post(api+"/devices/activate", "application/json", bytes.NewReader(body))
	if err != nil {
		fmt.Println("activate err:", err)
		return
	}
	resp.Body.Close()
	fmt.Println("device activated:", fp.DeviceID)
}

func heartbeat(api string, fp agentlib.Fingerprint) {
	ticker := time.NewTicker(60 * time.Second)
	client := &http.Client{Timeout: 5 * time.Second}
	for range ticker.C {
		t := store.Telemetry{
			DeviceID:  fp.DeviceID,
			LatencyMs: 12,
			MT4Conn:   true,
			MT5Conn:   true,
			Version:   "pat-engine-agent/0.1",
			Status:    "online",
		}
		body, _ := json.Marshal(t)
		resp, err := client.Post(api+"/devices/heartbeat", "application/json", bytes.NewReader(body))
		if err != nil {
			continue
		}
		resp.Body.Close()
	}
}
