package license

import (
	"strings"
	"testing"
	"time"
)

func TestSignParseRoundTrip(t *testing.T) {
	l, tok, err := DevLicense(DefaultDevSecret, []string{"ULTRA_SCALPING", "TREND_SWING"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Parse(tok, DefaultDevSecret)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got.Plan != l.Plan {
		t.Fatalf("plan mismatch: %s", got.Plan)
	}
	if !got.AllowsStrategy("ULTRA_SCALPING") || !got.AllowsStrategy("TREND_SWING") {
		t.Fatalf("allowed strategies not preserved")
	}
	if got.AllowsStrategy("STANDARD_SCALPING") {
		t.Fatalf("unlisted strategy should not be allowed")
	}
}

func TestSignatureTamperFails(t *testing.T) {
	_, tok, _ := DevLicense(DefaultDevSecret, []string{"*"}, nil)
	// flip a char in the signature portion
	if len(tok) < 4 {
		t.Fatal("token too short")
	}
	tampered := tok[:len(tok)-2] + "zz"
	if _, err := Parse(tampered, DefaultDevSecret); err == nil {
		t.Fatal("expected signature mismatch")
	}
}

func TestExpiry(t *testing.T) {
	l := &License{
		Key:               "X",
		Plan:              "PRO",
		AllowedStrategies: []string{"*"},
		ExpiresAt:         time.Now().Unix() - 10,
	}
	if err := l.IsValid(""); err == nil {
		t.Fatal("expired license should be invalid")
	}
	l.ExpiresAt = 0 // never
	if err := l.IsValid(""); err != nil {
		t.Fatalf("non-expiring license should be valid: %v", err)
	}
}

func TestDeviceBinding(t *testing.T) {
	l := &License{
		Key:               "X",
		Plan:              "PRO",
		AllowedStrategies: []string{"*"},
		DeviceID:          "device-123",
	}
	if err := l.IsValid("device-999"); err == nil {
		t.Fatal("device mismatch should be rejected")
	}
	if err := l.IsValid("device-123"); err != nil {
		t.Fatalf("matching device should be valid: %v", err)
	}
}

func TestWildcardAllowsAll(t *testing.T) {
	_, tok, _ := DevLicense(DefaultDevSecret, nil, nil) // dev with "*"
	l, _ := Parse(tok, DefaultDevSecret)
	if !l.AllowsStrategy("ANY_STRATEGY_AT_ALL") {
		t.Fatal("wildcard license should allow any strategy")
	}
}

func TestCompactRoundTrip(t *testing.T) {
	yes := true
	l := &License{
		Key:               "X",
		Plan:              "PRO",
		AllowedStrategies: []string{"ULTRA_SCALPING", "TREND_SWING"},
		ExpiresAt:         compactEpochBase + 365*86400,
		BrokerScalping:    &yes,
	}
	code, err := CompactSign(l, DefaultDevSecret)
	if err != nil {
		t.Fatal(err)
	}
	if len(code) > 30 {
		t.Fatalf("compact code too long: %q (%d chars)", code, len(code))
	}
	if !strings.HasPrefix(code, "PAT1-") {
		t.Fatalf("compact code missing prefix: %q", code)
	}
	got, err := Parse(code, DefaultDevSecret)
	if err != nil {
		t.Fatalf("parse compact: %v", err)
	}
	if got.Plan != "PRO" {
		t.Fatalf("plan mismatch: %s", got.Plan)
	}
	if !got.AllowsStrategy("ULTRA_SCALPING") || !got.AllowsStrategy("TREND_SWING") {
		t.Fatal("strategies not preserved")
	}
	if got.AllowsStrategy("STANDARD_SCALPING") {
		t.Fatal("unlisted strategy should not be allowed")
	}
	if got.BrokerScalping == nil || !*got.BrokerScalping {
		t.Fatal("broker scalping not preserved")
	}
}

func TestCompactTamperFails(t *testing.T) {
	code, _ := CompactSign(&License{Plan: "PRO", AllowedStrategies: []string{"*"}}, DefaultDevSecret)
	// flip last char
	tampered := code[:len(code)-1] + "X"
	if _, err := Parse(tampered, DefaultDevSecret); err == nil {
		t.Fatal("expected compact signature mismatch")
	}
}
