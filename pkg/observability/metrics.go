package observability

import (
	"context"
	"fmt"
	"strings"
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
	FetchResponseSize   metric.Float64Histogram
	FetchBytesTotal     metric.Int64Counter

	// Content processing metrics
	ContentProcessingDuration metric.Float64Histogram
	ContentSizeRatio          metric.Float64Histogram
	HTMLToMarkdownCount       metric.Int64Counter
	RobotsCheckCount          metric.Int64Counter
	RobotsBlockedCount        metric.Int64Counter

	// MCP protocol metrics
	MCPRequestDuration  metric.Float64Histogram
	MCPToolCallCount    metric.Int64Counter
	MCPSessionCount     metric.Int64UpDownCounter
	MCPErrorCount       metric.Int64Counter

	// Cache metrics
	CacheHits           metric.Int64Counter
	CacheMisses         metric.Int64Counter
	CacheEvictions      metric.Int64Counter
	CacheSize           metric.Int64UpDownCounter

	// Rate limiting metrics
	RateLimitHits       metric.Int64Counter
	ConcurrentRequests  metric.Int64UpDownCounter

	// Domain and error tracking metrics
	FetchByDomain       metric.Int64Counter
	RobotsViolations    metric.Int64Counter
	NetworkErrors       metric.Int64Counter
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

	// Fetch response size histogram
	metrics.FetchResponseSize, err = meter.Float64Histogram(
		"fetch_response_size_bytes",
		metric.WithDescription("Size of fetched responses in bytes"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create fetch_response_size_bytes metric: %w", err)
	}

	// Total bytes fetched counter
	metrics.FetchBytesTotal, err = meter.Int64Counter(
		"fetch_bytes_total",
		metric.WithDescription("Total bytes fetched"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create fetch_bytes_total metric: %w", err)
	}

	// Content size ratio histogram (markdown size / html size)
	metrics.ContentSizeRatio, err = meter.Float64Histogram(
		"content_size_ratio",
		metric.WithDescription("Ratio of markdown to HTML content size"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create content_size_ratio metric: %w", err)
	}

	// HTML to Markdown conversion counter
	metrics.HTMLToMarkdownCount, err = meter.Int64Counter(
		"html_to_markdown_conversions_total",
		metric.WithDescription("Total number of HTML to Markdown conversions"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create html_to_markdown_conversions_total metric: %w", err)
	}

	// Robots blocked counter
	metrics.RobotsBlockedCount, err = meter.Int64Counter(
		"robots_blocked_total",
		metric.WithDescription("Total number of requests blocked by robots.txt"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create robots_blocked_total metric: %w", err)
	}

	// MCP request duration histogram
	metrics.MCPRequestDuration, err = meter.Float64Histogram(
		"mcp_request_duration_seconds",
		metric.WithDescription("Duration of MCP protocol requests in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create mcp_request_duration_seconds metric: %w", err)
	}

	// MCP tool call counter
	metrics.MCPToolCallCount, err = meter.Int64Counter(
		"mcp_tool_calls_total",
		metric.WithDescription("Total number of MCP tool calls"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create mcp_tool_calls_total metric: %w", err)
	}

	// MCP session gauge
	metrics.MCPSessionCount, err = meter.Int64UpDownCounter(
		"mcp_active_sessions",
		metric.WithDescription("Number of active MCP sessions"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create mcp_active_sessions metric: %w", err)
	}

	// MCP error counter
	metrics.MCPErrorCount, err = meter.Int64Counter(
		"mcp_errors_total",
		metric.WithDescription("Total number of MCP protocol errors"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create mcp_errors_total metric: %w", err)
	}

	// Cache hit counter
	metrics.CacheHits, err = meter.Int64Counter(
		"cache_hits_total",
		metric.WithDescription("Total number of cache hits"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache_hits_total metric: %w", err)
	}

	// Cache miss counter
	metrics.CacheMisses, err = meter.Int64Counter(
		"cache_misses_total",
		metric.WithDescription("Total number of cache misses"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache_misses_total metric: %w", err)
	}

	// Cache eviction counter
	metrics.CacheEvictions, err = meter.Int64Counter(
		"cache_evictions_total",
		metric.WithDescription("Total number of cache evictions"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache_evictions_total metric: %w", err)
	}

	// Cache size gauge
	metrics.CacheSize, err = meter.Int64UpDownCounter(
		"cache_size_entries",
		metric.WithDescription("Current number of entries in cache"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache_size_entries metric: %w", err)
	}

	// Rate limit hits counter
	metrics.RateLimitHits, err = meter.Int64Counter(
		"rate_limit_hits_total",
		metric.WithDescription("Total number of rate limit hits"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create rate_limit_hits_total metric: %w", err)
	}

	// Concurrent requests gauge
	metrics.ConcurrentRequests, err = meter.Int64UpDownCounter(
		"concurrent_requests",
		metric.WithDescription("Number of concurrent requests being processed"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create concurrent_requests metric: %w", err)
	}

	// Fetch requests by domain counter
	metrics.FetchByDomain, err = meter.Int64Counter(
		"fetch_requests_by_domain",
		metric.WithDescription("Number of fetch requests per domain"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create fetch_requests_by_domain metric: %w", err)
	}

	// Robots.txt violations counter
	metrics.RobotsViolations, err = meter.Int64Counter(
		"robots_txt_violations_total",
		metric.WithDescription("Total number of robots.txt violations (blocked requests)"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create robots_txt_violations_total metric: %w", err)
	}

	// Network errors by type counter
	metrics.NetworkErrors, err = meter.Int64Counter(
		"network_errors_by_type",
		metric.WithDescription("Network errors categorized by type"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create network_errors_by_type metric: %w", err)
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

// RecordFetchResponseSize records the size of a fetched response
func (m *Metrics) RecordFetchResponseSize(ctx context.Context, url string, sizeBytes int64) {
	attrs := []attribute.KeyValue{
		attribute.String("url_host", extractHost(url)),
	}
	
	m.FetchResponseSize.Record(ctx, float64(sizeBytes), metric.WithAttributes(attrs...))
	m.FetchBytesTotal.Add(ctx, sizeBytes, metric.WithAttributes(attrs...))
}

// RecordContentSizeRatio records the size ratio between processed and original content
func (m *Metrics) RecordContentSizeRatio(ctx context.Context, originalSize, processedSize int) {
	if originalSize > 0 {
		ratio := float64(processedSize) / float64(originalSize)
		m.ContentSizeRatio.Record(ctx, ratio)
	}
}

// RecordHTMLToMarkdownConversion records an HTML to Markdown conversion
func (m *Metrics) RecordHTMLToMarkdownConversion(ctx context.Context, success bool) {
	attrs := []attribute.KeyValue{
		attribute.Bool("success", success),
	}
	m.HTMLToMarkdownCount.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordRobotsBlocked records when a request is blocked by robots.txt
func (m *Metrics) RecordRobotsBlocked(ctx context.Context, url string) {
	attrs := []attribute.KeyValue{
		attribute.String("url_host", extractHost(url)),
	}
	m.RobotsBlockedCount.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordMCPRequest records metrics for an MCP protocol request
func (m *Metrics) RecordMCPRequest(ctx context.Context, method string, duration time.Duration, success bool) {
	attrs := []attribute.KeyValue{
		attribute.String("method", method),
		attribute.Bool("success", success),
	}
	
	m.MCPRequestDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
	
	if !success {
		m.MCPErrorCount.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

// RecordMCPToolCall records an MCP tool call
func (m *Metrics) RecordMCPToolCall(ctx context.Context, toolName string) {
	attrs := []attribute.KeyValue{
		attribute.String("tool", toolName),
	}
	m.MCPToolCallCount.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordMCPSessionChange records a change in active MCP sessions
func (m *Metrics) RecordMCPSessionChange(ctx context.Context, delta int64) {
	m.MCPSessionCount.Add(ctx, delta)
}

// RecordCacheHit records a cache hit
func (m *Metrics) RecordCacheHit(ctx context.Context, cacheType string) {
	attrs := []attribute.KeyValue{
		attribute.String("cache_type", cacheType),
	}
	m.CacheHits.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordCacheMiss records a cache miss
func (m *Metrics) RecordCacheMiss(ctx context.Context, cacheType string) {
	attrs := []attribute.KeyValue{
		attribute.String("cache_type", cacheType),
	}
	m.CacheMisses.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordCacheEviction records a cache eviction
func (m *Metrics) RecordCacheEviction(ctx context.Context, cacheType string, count int64) {
	attrs := []attribute.KeyValue{
		attribute.String("cache_type", cacheType),
	}
	m.CacheEvictions.Add(ctx, count, metric.WithAttributes(attrs...))
}

// RecordCacheSizeChange records a change in cache size
func (m *Metrics) RecordCacheSizeChange(ctx context.Context, delta int64) {
	m.CacheSize.Add(ctx, delta)
}

// RecordRateLimitHit records when a rate limit is hit
func (m *Metrics) RecordRateLimitHit(ctx context.Context, limitType string) {
	attrs := []attribute.KeyValue{
		attribute.String("limit_type", limitType),
	}
	m.RateLimitHits.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordConcurrentRequestsChange records a change in concurrent requests
func (m *Metrics) RecordConcurrentRequestsChange(ctx context.Context, delta int64) {
	m.ConcurrentRequests.Add(ctx, delta)
}

// extractHost extracts the host from a URL for metrics labeling
func extractHost(urlStr string) string {
	// Parse URL to extract host
	if urlStr == "" {
		return "unknown"
	}

	// Find the start of the host after the protocol
	start := 0
	if idx := strings.Index(urlStr, "://"); idx != -1 {
		start = idx + 3
	}

	// Find the end of the host (before path or port)
	end := len(urlStr)
	for i := start; i < len(urlStr); i++ {
		if urlStr[i] == '/' || urlStr[i] == ':' || urlStr[i] == '?' {
			end = i
			break
		}
	}

	if start >= end {
		return "unknown"
	}

	host := urlStr[start:end]

	// Limit cardinality by returning "other" for very rare domains
	if len(host) > 100 {
		return "other"
	}

	return host
}

// RecordDomainFetch records fetch metrics per domain
func (m *Metrics) RecordDomainFetch(ctx context.Context, url string) {
	if m == nil {
		return
	}

	domain := extractHost(url)

	// Record fetch by domain
	attrs := []attribute.KeyValue{
		attribute.String("domain", domain),
	}
	m.FetchByDomain.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordRobotsViolation records when a request is blocked by robots.txt
func (m *Metrics) RecordRobotsViolation(ctx context.Context, url string) {
	domain := extractHost(url)
	attrs := []attribute.KeyValue{
		attribute.String("domain", domain),
	}
	m.RobotsViolations.Add(ctx, 1, metric.WithAttributes(attrs...))

	// Also record it in the general robots blocked counter
	m.RecordRobotsBlocked(ctx, url)
}

// RecordNetworkError records network errors by type
func (m *Metrics) RecordNetworkError(ctx context.Context, url string, errorType string) {
	domain := extractHost(url)
	attrs := []attribute.KeyValue{
		attribute.String("domain", domain),
		attribute.String("error_type", errorType),
	}
	m.NetworkErrors.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// ClassifyNetworkError classifies network errors into categories
func ClassifyNetworkError(err error) string {
	if err == nil {
		return "none"
	}

	errStr := err.Error()
	switch {
	case strings.Contains(errStr, "timeout"):
		return "timeout"
	case strings.Contains(errStr, "connection refused"):
		return "connection_refused"
	case strings.Contains(errStr, "no such host") || strings.Contains(errStr, "DNS"):
		return "dns_failure"
	case strings.Contains(errStr, "certificate") || strings.Contains(errStr, "x509"):
		return "tls_error"
	case strings.Contains(errStr, "reset by peer"):
		return "connection_reset"
	case strings.Contains(errStr, "broken pipe"):
		return "broken_pipe"
	case strings.Contains(errStr, "EOF"):
		return "unexpected_eof"
	default:
		return "other"
	}
}

// GetGlobalMetrics returns the global metrics instance
// This is a convenience function to get metrics from the global meter provider
func GetGlobalMetrics(serviceName string) (*Metrics, error) {
	return NewMetrics(otel.GetMeterProvider(), serviceName)
}