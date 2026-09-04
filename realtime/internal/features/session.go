package features

import (
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// liveBrokerOffsetHours holds the broker UTC offset observed live from the
// Master Node (EA reports TimeGMTOffset()/wall-clock diff on every tick).
// When non-zero it OVERRIDES the BROKER_TIMEZONE/BROKER_UTC_OFFSET defaults so
// hour-of-day logic always matches the exact clock the trading EAs run on
// (TimeCurrent). Wired from main.go via SetLiveBrokerOffset.
var liveBrokerOffsetHours atomic.Int32

// SetLiveBrokerOffset records the broker UTC offset (hours) reported live by
// the Master Node. It takes precedence over the static BROKER_TIMEZONE env so
// session classification never drifts from the broker's actual clock, even
// when the operator's static config is stale (e.g. a DST-observing broker).
func SetLiveBrokerOffset(hours int) {
	if hours < -12 || hours > 14 {
		return
	}
	liveBrokerOffsetHours.Store(int32(hours))
}

// LiveBrokerOffset returns the live-reported offset (0 = none observed yet).
func LiveBrokerOffset() int { return int(liveBrokerOffsetHours.Load()) }

// BrokerLocation returns the broker's operational timezone. Precedence:
//  1. Live offset observed from the Master Node (authoritative — matches the
//     exact clock the EAs' TimeCurrent() runs on).
//  2. BROKER_TIMEZONE env (IANA name or fixed "+HH"/"-HH" offset).
//  3. Fixed GMT+2 default (this deployment's XAUUSD broker server time, no DST).
//
// All time-of-day logic (session classification, ORB ranges, candle/signal
// timestamps that represent broker wall-clock time) MUST use this location,
// NOT UTC, otherwise every session boundary and displayed time is shifted.
// Absolute instants are still stored as TIMESTAMPTZ (UTC) in Postgres; only
// hour-of-day logic converts to this location.
func BrokerLocation() *time.Location {
	if off := LiveBrokerOffset(); off != 0 {
		return time.FixedZone(brokerZoneName(off), off*3600)
	}
	if tz := os.Getenv("BROKER_TIMEZONE"); tz != "" {
		if l, err := time.LoadLocation(tz); err == nil {
			return l
		}
		// Fixed-offset form: "+2", "-5", "+02:30"
		if off, ok := parseFixedOffsetHours(tz); ok {
			return time.FixedZone(brokerZoneName(off), off*3600)
		}
	}
	// Default: GMT+2 — this deployment's XAUUSD broker server time (fixed,
	// no DST for the fixed-offset broker session model).
	return time.FixedZone("GMT+2", 2*3600)
}

// brokerZoneName renders a display name for a fixed broker offset zone.
func brokerZoneName(offHours int) string {
	sign := "+"
	if offHours < 0 {
		sign = "-"
		offHours = -offHours
	}
	return "GMT" + sign + strconv.Itoa(offHours)
}

// parseFixedOffsetHours accepts "+2", "-5", "+02:30" style fixed offsets.
func parseFixedOffsetHours(s string) (int, bool) {
	if len(s) < 2 || (s[0] != '+' && s[0] != '-') {
		return 0, false
	}
	sign := 1
	if s[0] == '-' {
		sign = -1
	}
	body := s[1:]
	for i := 0; i < len(body); i++ {
		if body[i] == ':' {
			body = body[:i]
			break
		}
	}
	if body == "" {
		return 0, false
	}
	h := 0
	for i := 0; i < len(body); i++ {
		if body[i] < '0' || body[i] > '9' {
			return 0, false
		}
		h = h*10 + int(body[i]-'0')
	}
	if h > 14 {
		return 0, false
	}
	return sign * h, true
}

// SessionEngine determines the current trading session and news state.
// SOW Section 12D: Session/calendar/news state
// Sessions are classified in BROKER time (GMT+3), since the broker server time
// is GMT+3. The wall-clock boundaries below are expressed in broker-local time.
// Tokyo: 00:00-09:00 (no DST)
// London: 08:00-17:00 (DST shifts by ~1hr, handled by market calendar)
// New York: 13:00-22:00 (DST shifts by ~1hr)
// Overlap: 13:00-17:00 (London+NY)
// Sydney: 22:00-07:00
// NewsRiskProvider abstracts the economic-calendar risk engine so the features
// package does not depend on pkg/news directly. When nil (or when the provider
// reports the news provider as disabled), the session engine falls back to
// NewsRisk="NONE" — preserving the pre-v1.10 behaviour for operators who have
// not yet configured an economic-calendar provider.
type NewsRiskProvider interface {
	// ComputeNewsRisk returns the current news risk level string
	// (NONE/LOW/MEDIUM/HIGH/EXTREME/DATA_UNAVAILABLE).
	ComputeNewsRisk(now time.Time) string
}

type SessionEngine struct {
	location         *time.Location
	newsRiskProvider NewsRiskProvider
}

func NewSessionEngine() *SessionEngine {
	return &SessionEngine{location: BrokerLocation()}
}

// SetNewsRiskProvider injects an economic-calendar risk provider.
// Pass nil to disable news-risk computation (falls back to "NONE").
func (e *SessionEngine) SetNewsRiskProvider(p NewsRiskProvider) {
	e.newsRiskProvider = p
}

func (e *SessionEngine) Process(now time.Time) SessionFeatures {
	feat := SessionFeatures{}
	// Classify in BROKER time (GMT+3), not UTC — the broker server clock is GMT+3,
	// so session boundaries and weekend rollover must use broker-local time.
	brokerNow := now.In(e.location)
	weekday := brokerNow.Weekday()
	feat.IsWeekend = weekday == time.Saturday || weekday == time.Sunday

	// Session detection (BROKER-local hours, GMT+3)
	// TOKYO: 00:00-09:00 (no DST — Tokyo does not observe DST)
	// LONDON: 08:00-17:00 (07:00-16:00 during UK DST, handled by calendar)
	// NEW_YORK: 13:00-22:00 (12:00-21:00 during US DST)
	// OVERLAP: max(London start, NY start) to min(London end, NY end)
	// SYDNEY: 22:00-07:00
	hour := brokerNow.Hour()

	switch {
	case hour >= 0 && hour < 8:
		feat.CurrentSession = "TOKYO"
	case hour >= 8 && hour < 13:
		feat.CurrentSession = "LONDON"
	case hour >= 13 && hour < 17:
		feat.CurrentSession = "OVERLAP"
		feat.IsOverlap = true
	case hour >= 17 && hour < 22:
		feat.CurrentSession = "NEW_YORK"
	case hour >= 22:
		feat.CurrentSession = "SYDNEY"
	default:
		feat.CurrentSession = "OFF_HOURS"
	}

	// News risk: use the economic-calendar risk engine when configured.
	// When no provider is wired (nil) the fallback is "NONE" — this preserves
	// the pre-v1.10 behaviour for operators who have not yet configured a
	// news provider. Once a provider is configured, the RiskEngine returns
	// the real computed level (including DATA_UNAVAILABLE when it fails).
	if e.newsRiskProvider != nil {
		feat.NewsRisk = e.newsRiskProvider.ComputeNewsRisk(brokerNow)
	} else {
		feat.NewsRisk = "NONE"
	}

	return feat
}

// IsSessionAllowed checks if a strategy is allowed to trade in the current session.
// SOW Section 12D: Session-aware strategy eligibility.
// TOKYO is a first-class supported session for all strategies.
// Each strategy applies its own session-specific thresholds internally.
func IsSessionAllowed(strategy string, session string, isWeekend bool) bool {
	if isWeekend {
		return false
	}

	switch session {
	case "OFF_HOURS":
		return false
	case "TOKYO", "LONDON", "NEW_YORK", "OVERLAP", "SYDNEY",
	     "LONDON_NEWYORK_OVERLAP", "TOKYO_LONDON_OVERLAP", "LONDON_TOKYO_OVERLAP",
	     "NEWYORK_SYDNEY_OVERLAP", "SYDNEY_TOKYO_OVERLAP":
		// All active sessions and overlaps are eligible for all strategies.
		// Strategy-specific session thresholds (spread, ATR, cost) are
		// enforced inside each strategy's Evaluate() method.
		return true
	default:
		// Also accept any session containing "OVERLAP" or known session names
		// to handle agent-reported session names that may use different naming conventions
		upperSession := strings.ToUpper(session)
		if strings.Contains(upperSession, "OVERLAP") ||
			strings.Contains(upperSession, "LONDON") ||
			strings.Contains(upperSession, "NEW_YORK") ||
			strings.Contains(upperSession, "NEWYORK") ||
			strings.Contains(upperSession, "TOKYO") ||
			strings.Contains(upperSession, "SYDNEY") {
			return true
		}
		return false
	}
}
