package broker

import (
	"testing"
	"time"
)

func TestExecutionRoundingAndCost(t *testing.T) {
	e := DefaultXAUUSDExecution()
	if got := e.RoundToDigits(2000.456); got != 2000.46 {
		t.Fatalf("RoundToDigits = %v, want 2000.46", got)
	}
	if e.Points(0.20) != 20 {
		t.Fatalf("Points(0.20) = %v, want 20", e.Points(0.20))
	}
	// $7 commission per lot, contract 100 => 0.07 price units
	cp := e.CommissionPrice(1.0)
	if cp < 0.069 || cp > 0.071 {
		t.Fatalf("CommissionPrice = %v, want ~0.07", cp)
	}
	// required margin: 1 lot * 100 * 2000 / 500 = 400
	if m := e.RequiredMargin(1.0, 2000.0); m != 400 {
		t.Fatalf("RequiredMargin = %v, want 400", m)
	}
	if ma := e.MaxAffordableLot(400, 2000.0); ma != 1.0 {
		t.Fatalf("MaxAffordableLot = %v, want 1.0", ma)
	}
}

func TestBrokerSessionTimezone(t *testing.T) {
	e := DefaultXAUUSDExecution() // broker UTC+4
	pol := &BrokerPolicy{Execution: e}

	// UTC 09:30 -> broker 13:30 -> London/NY overlap
	n1 := time.Date(2026, 1, 1, 9, 30, 0, 0, time.UTC)
	if name, overlap := pol.Session(n1); name != "OVERLAP" || !overlap {
		t.Fatalf("Session(09:30 UTC) = %s overlap=%v, want OVERLAP true", name, overlap)
	}

	// UTC 22:00 -> broker 02:00 -> Tokyo/Asia session
	n2 := time.Date(2026, 1, 1, 22, 0, 0, 0, time.UTC)
	if name, overlap := pol.Session(n2); name != "TOKYO" || overlap {
		t.Fatalf("Session(22:00 UTC) = %s overlap=%v, want TOKYO false", name, overlap)
	}
}
