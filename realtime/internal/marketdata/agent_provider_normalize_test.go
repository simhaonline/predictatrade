package marketdata

import "testing"

func TestNormalizeSymbol_GlobalBrokerVariants(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"XAUUSD", "XAUUSD"},
		{"XAUUSD.sd", "XAUUSD"},
		{"XAUUSD.e", "XAUUSD"},
		{"XAUUSD.m", "XAUUSD"},
		{"XAUUSDpro", "XAUUSD"},
		{"XAU/USD", "XAUUSD"},
		{"XAU USD", "XAUUSD"},
		{"xauusd.sd", "XAUUSD"},
		{"GOLD", "XAUUSD"},
		{"GOLD.sb", "XAUUSD"},
		// Non-gold instruments are left untouched (rejected downstream).
		{"EURUSD", "EURUSD"},
		{"BTCUSD", "BTCUSD"},
		{"US30.cash", "US30.cash"},
	}
	for _, c := range cases {
		if got := normalizeSymbol(c.in); got != c.want {
			t.Errorf("normalizeSymbol(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
