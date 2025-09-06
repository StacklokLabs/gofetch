package observability

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// PrometheusHandler returns an HTTP handler for Prometheus metrics scraping
func (t *Telemetry) PrometheusHandler() http.Handler {
	// When using OpenTelemetry Prometheus exporter with a MeterProvider,
	// the metrics are automatically registered with the global Prometheus registry.
	// We can use the standard promhttp.Handler() for this case.
	return promhttp.Handler()
}

// GetPrometheusHandler returns a standalone Prometheus handler
// This can be used if you only want Prometheus metrics without full OpenTelemetry setup
func GetPrometheusHandler() (http.Handler, error) {
	// For a basic Prometheus endpoint, use the standard handler
	return promhttp.Handler(), nil
}