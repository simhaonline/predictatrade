package strategy

import (
	"testing"
)

func TestCheckAllStrategies(t *testing.T) {
	strats := AllStrategies()
	t.Logf("Total strategies: %d", len(strats))
	for i, s := range strats {
		t.Logf("  %d: %s", i, s.ID())
	}
}
