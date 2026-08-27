package features

import (
	"sync"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

// TestStateManagerConcurrentGetUpdateClone guards the regression fixed by the
// snapshot-clone in StateManager.Get: concurrent Update (writer) and Get/Clone
// (reader) on the shared MarketState must not race. Without the clone, readers
// touched the live pointer while writers mutated it, corrupting signal state and
// (under -race) failing. Run with: go test -race.
func TestStateManagerConcurrentGetUpdateClone(t *testing.T) {
	sm := NewStateManager()
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 300; j++ {
				sm.Update("XAUUSD", func(s *MarketState) {
					s.Timestamp = time.Now().UTC()
					s.Session.CurrentSession = "LONDON"
					s.Spread = s.Spread.Add(decimal.NewFromFloat(0.01))
					s.Structure.SwingLows = append(s.Structure.SwingLows, decimal.NewFromFloat(float64(j)))
				})
				s := sm.Get("XAUUSD")
				c := s.Clone()
				_ = c.Session.CurrentSession
				_ = c.Structure.SwingLows
			}
		}()
	}
	wg.Wait()

	// Final read must be consistent (no corrupt/partial state).
	final := sm.Get("XAUUSD").Clone()
	if final.Symbol != "XAUUSD" {
		t.Fatalf("unexpected symbol %q", final.Symbol)
	}
}
