package igs

import "time"

// CrossMarketDriver is the minimal structural subset of
// crossmarket.DriverSnapshot that the fan-in consumes. Declared locally to
// avoid an import cycle; the adapter in main.go maps real snapshots.
type CrossMarketDriver struct {
	Name        string
	RawValue    float64
	ImpactScore float64
	Confidence  float64
	Quality     string
	Source      string
	Reason      string
	Timestamp   time.Time
}

// QualityFromCrossQuality maps crossmarket DataQuality strings to IGS Quality.
func QualityFromCrossQuality(q string) Quality {
	switch q {
	case "CONNECTED":
		return QualityConnected
	case "DEGRADED":
		return QualityDegraded
	case "STALE":
		return QualityStale
	case "ERROR":
		return QualityError
	default:
		return QualityUnavailable
	}
}
