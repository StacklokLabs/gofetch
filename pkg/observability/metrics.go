package observability

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Metrics holds all the application metrics
type Metrics struct {
	// HTTP metrics
	HTTPRequestDuration metric.Float64Histogram
	HTTPRequestCount    metric.Int64Counter
	HTTPActiveRequests  metric.Int64UpDownCounter

	// Fetch operation metrics
	FetchDuration       metric.Float64Histogram
	FetchCount          metric.Int64Counter
	FetchErrors         metric.Int64Counter

	// Content processing metrics
	ContentProcessingDuration metric.Float64Histogram
	RobotsCheckCount          metric.Int64Counter
}

// NewMetrics creates and registers all application metrics
func NewMetrics(meterProvider metric.MeterProvider, serviceName string) (*Metrics, error) {
	meter := meterProvider.Meter(serviceName)

	metrics := &Metrics{}

	var err error

	// HTTP request duration histogram
	metrics.HTTPRequestDuration, err = meter.Float64Histogram(
		"http_request_duration_seconds",
		metric.WithDescription("Duration of HTTP requests in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create http_request_duration_seconds metric: %w", err)
	}

	// HTTP request counter
	metrics.HTTPRequestCount, err = meter.Int64Counter(
		"http_requests_total",
		metric.WithDescription("Total number of HTTP requests"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create http_requests_total metric: %w", err)
	}

	// HTTP active requests gauge
	metrics.HTTPActiveRequests, err = meter.Int64UpDownCounter(
		"http_active_requests",
		metric.WithDescription("Number of active HTTP requests"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create http_active_requests metric: %w", err)
	}

	// Fetch operation duration histogram
	metrics.FetchDuration, err = meter.Float64Histogram(
		"fetch_duration_seconds",
		metric.WithDescription("Duration of fetch operations in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create fetch_duration_seconds metric: %w", err)
	}

	// Fetch operation counter
	metrics.FetchCount, err = meter.Int64Counter(
		"fetch_operations_total",
		metric.WithDescription("Total number of fetch operations"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create fetch_operations_total metric: %w", err)
	}

	// Fetch errors counter
	metrics.FetchErrors, err = meter.Int64Counter(
		"fetch_errors_total",
		metric.WithDescription("Total number of fetch operation errors"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create fetch_errors_total metric: %w", err)
	}

	// Content processing duration histogram
	metrics.ContentProcessingDuration, err = meter.Float64Histogram(
		"content_processing_duration_seconds",
		metric.WithDescription("Duration of content processing in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create content_processing_duration_seconds metric: %w", err)
	}

	// Robots.txt check counter
	metrics.RobotsCheckCount, err = meter.Int64Counter(
		"robots_check_total",
		metric.WithDescription("Total number of robots.txt checks"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create robots_check_total metric: %w", err)
	}

	return metrics, nil
}

// RecordHTTPRequest records metrics for an HTTP request
func (m *Metrics) RecordHTTPRequest(ctx context.Context, method, path, statusCode string, duration time.Duration) {
	attrs := []attribute.KeyValue{
		attribute.String("method", method),
		attribute.String("path", path),
		attribute.String("status_code", statusCode),
	}

	m.HTTPRequestDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
	m.HTTPRequestCount.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordHTTPActiveRequestsChange records a change in active HTTP requests
func (m *Metrics) RecordHTTPActiveRequestsChange(ctx context.Context, delta int64) {
	m.HTTPActiveRequests.Add(ctx, delta)
}

// RecordFetchOperation records metrics for a fetch operation
func (m *Metrics) RecordFetchOperation(ctx context.Context, url, result string, duration time.Duration) {
	attrs := []attribute.KeyValue{
		attribute.String("url_host", extractHost(url)),
		attribute.String("result", result), // "success" or "error"
	}

	m.FetchDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
	m.FetchCount.Add(ctx, 1, metric.WithAttributes(attrs...))

	if result == "error" {
		m.FetchErrors.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// RecordContentProcessing records metrics for content processing
func (m *Metrics) RecordContentProcessing(ctx context.Context, contentType string, duration time.Duration) {
	attrs := []attribute.KeyValue{
		attribute.String("content_type", contentType),
	}

	m.ContentProcessingDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
}

// RecordRobotsCheck records metrics for a robots.txt check
func (m *Metrics) RecordRobotsCheck(ctx context.Context, url, result string) {
	attrs := []attribute.KeyValue{
		attribute.String("url_host", extractHost(url)),
		attribute.String("result", result), // "allowed" or "disallowed"
	}

	m.RobotsCheckCount.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// extractHost extracts the host from a URL for metrics labeling
func extractHost(urlStr string) string {
	// Simple host extraction - in production you might want more robust URL parsing
	if len(urlStr) > 50 {
		// For very long URLs, just use a generic label to avoid cardinality explosion
		return "long_url"
	}
	return urlStr
}

// GetGlobalMetrics returns the global metrics instance
// This is a convenience function to get metrics from the global meter provider
func GetGlobalMetrics(serviceName string) (*Metrics, error) {
	return NewMetrics(otel.GetMeterProvider(), serviceName)
}