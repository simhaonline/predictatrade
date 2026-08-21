// Package mlengine — Prometheus metrics for ML inference.
package mlengine

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// mlRequestsTotal counts total ML inference requests.
	mlRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ml_requests_total",
		Help: "Total ML inference requests",
	}, []string{"status"}) // status: success, error, disabled

	// mlLatencyHistogram tracks ML inference latency in milliseconds.
	mlLatencyHistogram = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ml_latency_histogram",
		Help:    "ML inference latency in milliseconds",
		Buckets: []float64{0.5, 1, 2, 5, 10, 15, 20, 50, 100},
	}, []string{"model"}) // model: xgb, lstm, ensemble

	// mlConfidenceGauge tracks the latest ML confidence level.
	mlConfidenceGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ml_confidence_gauge",
		Help: "Latest ML prediction confidence (0-100)",
	}, []string{"direction"}) // direction: BUY, HOLD, SELL

	// mlModelVersionInfo exposes the current model version.
	mlModelVersionInfo = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ml_model_version_info",
		Help: "ML model version info (value=1 when active)",
	}, []string{"version"}) // version: model version string
)

// recordMetrics updates Prometheus metrics after a prediction.
func recordMetrics(pred *Prediction, err error) {
	if err != nil {
		mlRequestsTotal.WithLabelValues("error").Inc()
		return
	}
	mlRequestsTotal.WithLabelValues("success").Inc()
	mlLatencyHistogram.WithLabelValues("ensemble").Observe(pred.LatencyMs)
	mlConfidenceGauge.WithLabelValues(pred.Direction).Set(pred.Confidence)
}

// setModelVersionMetric updates the model version gauge.
func setModelVersionMetric(version string) {
	mlModelVersionInfo.WithLabelValues(version).Set(1)
}
