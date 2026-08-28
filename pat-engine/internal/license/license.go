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
func Parse(token, secret string) (*License, error) {
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
