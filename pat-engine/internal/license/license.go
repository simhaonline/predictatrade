// Package license implements the standalone license / entitlement gate for the
// signal engine. It is intentionally self-contained (stdlib crypto only) and mirrors
// the control-plane contract: a license grants a PLAN and an explicit list of
// allowed_strategies. The engine enforces allowed_strategies server-side (never an
// EA input), exactly as required by the architecture boundaries.
//
// Tokens are HMAC-SHA256 signed (control plane = signer, engine = verifier). In
// production the signing secret is held by the NestJS control plane; the engine only
// verifies. A dev secret is provided so the engine runs without the control plane.
package license

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// DefaultDevSecret is used only when no control-plane secret is configured. It lets
// the engine run in isolation; real deployments MUST inject PAT_LICENSE_SECRET.
const DefaultDevSecret = "pat-dev-secret"

// License is the parsed entitlement.
type License struct {
	Key                 string   `json:"key"`
	Plan                string   `json:"plan"`
	AllowedStrategies   []string `json:"allowed_strategies"`
	DeviceID            string   `json:"device_id,omitempty"`
	ExpiresAt           int64    `json:"expires_at"` // unix seconds; 0 = never
	BrokerScalping      *bool    `json:"broker_scalping,omitempty"`
	Signature           string   `json:"signature,omitempty"`
}

// Sign returns a tamper-evident token: base64(json) + "." + hex(hmac(json)).
func Sign(l *License, secret string) (string, error) {
	raw, err := json.Marshal(l)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(raw)
	sig := hex.EncodeToString(mac.Sum(nil))
	return base64.StdEncoding.EncodeToString(raw) + "." + sig, nil
}

// Parse verifies the HMAC signature with the given secret and returns the license.
// It transparently accepts both the full signed token and the short, user-friendly
// compact activation code (PAT1-...). The compact code is what we circulate to end
// users; the long token remains the internal machine representation.
func Parse(token, secret string) (*License, error) {
	if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(token)), compactPrefix) {
		return CompactParse(token, secret)
	}
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return nil, errors.New("license: malformed token")
	}
	raw, err := base64.StdEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("license: bad payload: %w", err)
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(raw)
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(parts[1])) {
		return nil, errors.New("license: signature mismatch")
	}
	var l License
	if err := json.Unmarshal(raw, &l); err != nil {
		return nil, fmt.Errorf("license: unmarshal: %w", err)
	}
	return &l, nil
}

// IsValid checks signature-independent constraints: expiry and (optional) device.
// Signature is verified by Parse; callers should Parse first.
func (l *License) IsValid(deviceID string) error {
	if l.ExpiresAt > 0 && time.Now().Unix() > l.ExpiresAt {
		return errors.New("license: expired")
	}
	if l.DeviceID != "" && deviceID != "" && l.DeviceID != deviceID {
		return errors.New("license: device mismatch")
	}
	return nil
}

// AllowsStrategy reports whether the license entitles the given strategy.
func (l *License) AllowsStrategy(id string) bool {
	for _, a := range l.AllowedStrategies {
		if a == id || a == "*" {
			return true
		}
	}
	return false
}

// AllowsScalping resolves broker scalping permission: explicit license value wins;
// nil means "unspecified" and defers to the broker policy.
func (l *License) AllowsScalping() *bool { return l.BrokerScalping }

// DevLicense builds a signed dev token that allows the given strategies (or all if
// empty) and optionally overrides scalping. Useful for local runs without the
// control plane.
func DevLicense(secret string, strategies []string, scalping *bool) (*License, string, error) {
	if secret == "" {
		secret = DefaultDevSecret
	}
	if len(strategies) == 0 {
		strategies = []string{"*"}
	}
	l := &License{
		Key:               "DEV",
		Plan:              "DEV",
		AllowedStrategies: strategies,
		ExpiresAt:         0,
		BrokerScalping:    scalping,
	}
	tok, err := Sign(l, secret)
	if err != nil {
		return nil, "", err
	}
	return l, tok, nil
}

// Scalping lets callers build a *bool conveniently.
func Scalping(b bool) *bool { return &b }

// ---------------------------------------------------------------------------
// Compact activation code (user-facing, short, typeable)
//
// The full signed token is ~200 chars (base64 JSON + 64-hex HMAC) — great for a
// machine but painful to circulate. The compact code packs the entitlement into 5
// bytes (version+plan, strategy bitmap, expiry-in-days, scalping flag) plus a 6-byte
// (48-bit) HMAC truncated for tamper-evidence, base32-encoded and grouped:
//
//	PAT1-XXXX-XXXX-XXXX-XXXX-XXXX-XX   (~21 chars, Crockford-friendly A-Z2-7)
//
// 48-bit HMAC is sufficient for a license activation code verified server-side; it is
// not a substitute for the full token's 256-bit signature where that matters.
// ---------------------------------------------------------------------------

const compactPrefix = "PAT1-"

var planToCode = map[string]byte{"DEV": 0, "BASIC": 1, "PRO": 2, "ENTERPRISE": 3}
var codeToPlan = map[byte]string{0: "DEV", 1: "BASIC", 2: "PRO", 3: "ENTERPRISE"}

const (
	cStratUltra     byte = 1
	cStratStdScalp  byte = 2
	cStratStdSwing  byte = 4
	cStratTrend     byte = 8
	cStratAll       byte = 15
)

// compactEpochBase = 2024-01-01 UTC; expiry is stored as days since this base.
const compactEpochBase int64 = 1704067200

// CompactSign returns a short activation code for the given license.
func CompactSign(l *License, secret string) (string, error) {
	var b0 byte = (1 << 4) | planToCode[strings.ToUpper(l.Plan)]
	var sb byte
	for _, s := range l.AllowedStrategies {
		switch strings.ToUpper(s) {
		case "*", "":
			sb |= cStratAll
		case "ULTRA_SCALPING":
			sb |= cStratUltra
		case "STANDARD_SCALPING":
			sb |= cStratStdScalp
		case "STANDARD_SWING":
			sb |= cStratStdSwing
		case "TREND_SWING":
			sb |= cStratTrend
		}
	}
	if len(l.AllowedStrategies) == 0 {
		sb |= cStratAll
	}
	var exp uint16
	if l.ExpiresAt > 0 {
		d := (l.ExpiresAt - compactEpochBase) / 86400
		if d < 0 {
			d = 0
		}
		if d > 65535 {
			d = 65535
		}
		exp = uint16(d)
	}
	var sc byte
	if l.BrokerScalping != nil {
		if *l.BrokerScalping {
			sc = 1
		} else {
			sc = 2
		}
	}
	payload := []byte{b0, sb, byte(exp & 0xff), byte(exp >> 8), sc}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	sum := mac.Sum(nil)
	body := append(append([]byte{}, payload...), sum[:6]...)
	enc := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(body)
	var b strings.Builder
	b.WriteString(compactPrefix)
	for i := 0; i < len(enc); i++ {
		if i > 0 && i%4 == 0 {
			b.WriteByte('-')
		}
		b.WriteByte(enc[i])
	}
	return b.String(), nil
}

// CompactParse verifies a compact activation code and reconstructs the license.
func CompactParse(code, secret string) (*License, error) {
	c := strings.ToUpper(strings.ReplaceAll(code, "-", ""))
	if !strings.HasPrefix(c, "PAT1") {
		return nil, errors.New("license: not a compact code")
	}
	raw, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(c[len("PAT1"):])
	if err != nil {
		return nil, fmt.Errorf("license: bad compact payload: %w", err)
	}
	if len(raw) != 11 {
		return nil, errors.New("license: compact code wrong length")
	}
	payload, macB := raw[:5], raw[5:]
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	if !hmac.Equal(mac.Sum(nil)[:6], macB) {
		return nil, errors.New("license: compact signature mismatch")
	}
	version := payload[0] >> 4
	if version != 1 {
		return nil, errors.New("license: unsupported compact version")
	}
	plan := codeToPlan[payload[0]&0x0F]
	var strats []string
	sb := payload[1]
	if sb&cStratAll == cStratAll {
		strats = []string{"*"}
	} else {
		if sb&cStratUltra != 0 {
			strats = append(strats, "ULTRA_SCALPING")
		}
		if sb&cStratStdScalp != 0 {
			strats = append(strats, "STANDARD_SCALPING")
		}
		if sb&cStratStdSwing != 0 {
			strats = append(strats, "STANDARD_SWING")
		}
		if sb&cStratTrend != 0 {
			strats = append(strats, "TREND_SWING")
		}
	}
	var exp int64
	if payload[2] != 0 || payload[3] != 0 {
		d := int64(uint16(payload[2]) | uint16(payload[3])<<8)
		exp = compactEpochBase + d*86400
	}
	var sc *bool
	switch payload[4] {
	case 1:
		b := true
		sc = &b
	case 2:
		b := false
		sc = &b
	}
	return &License{
		Key:               "COMPACT",
		Plan:              plan,
		AllowedStrategies: strats,
		ExpiresAt:         exp,
		BrokerScalping:    sc,
	}, nil
}
