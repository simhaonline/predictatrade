package config

import "testing"

// helperDefault returns a default config with a valid DBURL for testing.
func helperDefault() *Config {
	c := Default()
	if c.DBURL == "" {
		c.DBURL = "postgresql://test:test@localhost/test"
	}
	return c
}

func TestValidateCapitalProtectionNesting(t *testing.T) {
	base := helperDefault()
	// Defaults are safe
	if err := base.Validate(); err != nil {
		t.Fatalf("default config must validate: %v", err)
	}

	cases := []struct {
		name    string
		mutate  func(*Config)
		wantErr bool
	}{
		{"daily < weekly < monthly ok", func(c *Config) { c.MaxDailyLossPct, c.MaxWeeklyLossPct, c.MaxMonthlyLossPct = 2, 4, 5 }, false},
		{"daily >= weekly rejected", func(c *Config) { c.MaxDailyLossPct, c.MaxWeeklyLossPct = 5, 4 }, true},
		{"weekly >= monthly rejected", func(c *Config) { c.MaxWeeklyLossPct, c.MaxMonthlyLossPct = 6, 5 }, true},
		{"profit lock below loss magnitude rejected", func(c *Config) { c.MaxDailyProfitPct = 1 }, true},
		{"risk pct over 100 rejected", func(c *Config) { c.MaxRiskPerTradePct = 150 }, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := helperDefault()
			tc.mutate(c)
			err := c.Validate()
			if tc.wantErr && err == nil {
				t.Error("expected validation error")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}

	// Entirely-unset block (manual Config literal) skips capital checks.
	legacy := &Config{WSPort: 13081, DBURL: "postgresql://user:pass@localhost/db"}
	if err := legacy.Validate(); err != nil {
		if containsCapitalProtection(err.Error()) {
			t.Errorf("legacy zero-value config must not fail capital checks: %v", err)
		}
	}
}

func containsCapitalProtection(msg string) bool {
	return len(msg) > len("capital protection") && msg[:19] == "capital protection:"
}

func TestEnableShortsDefaultTrue(t *testing.T) {
	if !Default().EnableShorts {
		t.Error("ENABLE_SHORTS must default to true")
	}
}
