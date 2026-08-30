package crossmarket

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Prometheus metrics for the Cross-Market Confluence Engine.
var (
	MacroDataAge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "macro_data_age_seconds",
		Help: "Age of macro data in seconds",
	}, []string{"driver"})

	MacroScore = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "macro_score",
		Help: "Current cross-market confluence score (-100 to +100)",
	})

	MacroConfidence = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "macro_confidence",
		Help: "Current cross-market confidence (0 to 1)",
	})

	MacroDataQuality = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "macro_data_quality",
		Help: "Cross-market data quality (0=missing, 1=connected)",
	})

	MacroRegime = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "macro_regime",
		Help: "Current macro regime (encoded as number)",
	})

	MacroSignalAdjustment = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "macro_signal_adjustment",
		Help: "Score adjustment applied to signal (0 in shadow mode)",
	})

	MacroProviderUp = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "macro_provider_up",
		Help: "Provider health (1=up, 0=down)",
	}, []string{"provider"})

)

// RecordMetrics updates Prometheus metrics from a confluence result.
func RecordMetrics(result *ConfluenceResult) {
	MacroScore.Set(result.Score)
	MacroConfidence.Set(result.Confidence)
	MacroSignalAdjustment.Set(result.ScoreAdjustment)

	qMap := map[DataQuality]float64{
		QualityConnected: 1.0,
		QualityDegraded:  0.5,
		QualityStale:     0.25,
		QualityMissing:   0.0,
		QualityError:     0.0,
	}
	MacroDataQuality.Set(qMap[result.DataQuality])

	regimeMap := map[SafeHavenRegime]float64{
		SHNORMAL: 0, SHRiskOn: 1, SHRiskOff: 2,
		SHSafeHavenGold: 3, SHSafeHavenUSD: 4, SHDualSafeHaven: 5,
		SHLiquidityStress: 6, SHMixed: 7, SHUnknown: -1,
	}
	MacroRegime.Set(regimeMap[result.Regime])

	for _, d := range result.DriverSnapshot {
		MacroDataAge.WithLabelValues(string(d.Name)).Set(0) // would be actual age in production
		MacroProviderUp.WithLabelValues(d.Source).Set(1)
	}
}
