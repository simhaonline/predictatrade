package features

import (
	"strings"
	"time"
)

// SessionEngine determines the current trading session and news state.
// SOW Section 12D: Session/calendar/news state
// Sessions are UTC-based with proper session boundaries.
// Tokyo: 00:00-09:00 UTC (no DST)
// London: 08:00-17:00 UTC (DST shifts by ~1hr, handled by market calendar)
// New York: 13:00-22:00 UTC (DST shifts by ~1hr)
// Overlap: 13:00-17:00 UTC (London+NY)
// Sydney: 22:00-07:00 UTC
type SessionEngine struct {
	location *time.Location
}

func NewSessionEngine() *SessionEngine {
	loc, _ := time.LoadLocation("UTC")
	return &SessionEngine{location: loc}
}

func (e *SessionEngine) Process(now time.Time) SessionFeatures {
	feat := SessionFeatures{}
	utc := now.UTC()

	// Weekend check (Saturday/Sunday UTC)
	weekday := utc.Weekday()
	feat.IsWeekend = weekday == time.Saturday || weekday == time.Sunday

	// Session detection (UTC hours) — production-grade session model
	// TOKYO: 00:00-09:00 UTC (no DST — Tokyo does not observe DST)
	// LONDON: 08:00-17:00 UTC (07:00-16:00 during UK DST, handled by calendar)
	// NEW_YORK: 13:00-22:00 UTC (12:00-21:00 during US DST)
	// OVERLAP: max(London start, NY start) to min(London end, NY end)
	// SYDNEY: 22:00-07:00 UTC
	hour := utc.Hour()

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

	// News risk: simplified (in production, query economic calendar)
	feat.NewsRisk = "NONE"

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
