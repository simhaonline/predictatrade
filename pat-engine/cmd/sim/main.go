package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"pat-engine/internal/provider"
)

// cmd/sim simulates the MQL EA consuming PAT_signals.txt. It polls the file, prints
// the trade the EA would execute, then deletes the file exactly like the EA does —
// proving the engine->signal-file->EA handoff end-to-end (without a live terminal).
func main() {
	path := "signals/PAT_signals.txt"
	fmt.Println("EA simulator watching", path)
	for {
		data, err := os.ReadFile(path)
		if err == nil && len(data) > 0 {
			line := strings.TrimSpace(string(data))
			if strings.HasPrefix(line, "SIGNAL|") {
				var dto provider.SignalDTO
				if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "SIGNAL|")), &dto); err == nil {
					fmt.Printf("[EA EXECUTE] %-16s %-5s @ %.2f SL=%.2f TP1=%.2f grade=%s score=%.1f\n",
						dto.StrategyID, dto.Direction, dto.EntryPrice, dto.StopLoss, dto.TP1, dto.Grade, dto.RawScore)
				}
			}
			os.Remove(path) // consume like the EA
		}
		time.Sleep(200 * time.Millisecond)
	}
}
