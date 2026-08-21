package types

import (
	"encoding/json"
	"testing"
	"github.com/shopspring/decimal"
)

func TestSignalJSON(t *testing.T) {
	signal := &Signal{
		ID:         "test-123",
		Symbol:     "XAUUSD",
		StrategyID: StrategyTrendSwing,
		Direction:  Direction("BUY_CANDIDATE"),
		Grade:      GradeResearch,
		RawScore:   decimal.NewFromFloat(41),
		EntryPrice: decimal.NewFromFloat(4519.03),
		StopLoss:   decimal.NewFromFloat(4473.11),
		TP1:        decimal.NewFromFloat(4564.53),
		TP2:        decimal.NewFromFloat(4610.17),
		TP3:        decimal.Zero,
	}
	
	data, err := json.Marshal(signal)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	t.Logf("JSON output:\n%s", string(data))
	
	// Check if key fields are present
	jsonStr := string(data)
	for _, key := range []string{"\"ID\"", "\"Direction\"", "\"EntryPrice\"", "\"StopLoss\"", "\"TP1\"", "\"TP2\""} {
		if !contains(jsonStr, key) {
			t.Errorf("JSON missing key: %s", key)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && indexOf(s, substr) >= 0)
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
