package observability

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
)

func TestNewMetrics(t *testing.T) {
	// Create a test meter provider
	res := resource.Default()
	meterProvider := metric.NewMeterProvider(
		metric.WithResource(res),
	)

	serviceName := "test-service"
	metrics, err := NewMetrics(meterProvider, serviceName)

	if err != nil {
		t.Errorf("expected no error creating metrics, got %v", err)
	}
	if metrics == nil {
		t.Fatal("expected metrics instance to be created")
	}

	// Check that all metrics are initialized
	if metrics.HTTPRequestDuration == nil {
		t.Error("expected HTTPRequestDuration to be initialized")
	}
	if metrics.HTTPRequestCount == nil {
		t.Error("expected HTTPRequestCount to be initialized")
	}
	if metrics.HTTPActiveRequests == nil {
		t.Error("expected HTTPActiveRequests to be initialized")
	}
	if metrics.FetchDuration == nil {
		t.Error("expected FetchDuration to be initialized")
	}
	if metrics.FetchCount == nil {
		t.Error("expected FetchCount to be initialized")
	}
	if metrics.FetchErrors == nil {
		t.Error("expected FetchErrors to be initialized")
	}
	if metrics.ContentProcessingDuration == nil {
		t.Error("expected ContentProcessingDuration to be initialized")
	}
	if metrics.RobotsCheckCount == nil {
		t.Error("expected RobotsCheckCount to be initialized")
	}
}

func TestRecordHTTPRequest(t *testing.T) {
	meterProvider := noop.NewMeterProvider()
	metrics, err := NewMetrics(meterProvider, "test-service")
	if err != nil {
		t.Fatalf("failed to create metrics: %v", err)
	}

	ctx := context.Background()
	duration := 100 * time.Millisecond

	// This should not panic
	metrics.RecordHTTPRequest(ctx, "GET", "/test", "200", duration)
}

func TestRecordFetchOperation(t *testing.T) {
	meterProvider := noop.NewMeterProvider()
	metrics, err := NewMetrics(meterProvider, "test-service")
	if err != nil {
		t.Fatalf("failed to create metrics: %v", err)
	}

	ctx := context.Background()
	duration := 500 * time.Millisecond

	// Test successful operation
	metrics.RecordFetchOperation(ctx, "https://example.com", "success", duration)

	// Test failed operation
	metrics.RecordFetchOperation(ctx, "https://invalid.com", "error", duration)
}

func TestRecordContentProcessing(t *testing.T) {
	meterProvider := noop.NewMeterProvider()
	metrics, err := NewMetrics(meterProvider, "test-service")
	if err != nil {
		t.Fatalf("failed to create metrics: %v", err)
	}

	ctx := context.Background()
	duration := 50 * time.Millisecond

	// This should not panic
	metrics.RecordContentProcessing(ctx, "text/html", duration)
}

func TestRecordRobotsCheck(t *testing.T) {
	meterProvider := noop.NewMeterProvider()
	metrics, err := NewMetrics(meterProvider, "test-service")
	if err != nil {
		t.Fatalf("failed to create metrics: %v", err)
	}

	ctx := context.Background()

	// Test allowed result
	metrics.RecordRobotsCheck(ctx, "https://example.com", "allowed")

	// Test disallowed result
	metrics.RecordRobotsCheck(ctx, "https://restricted.com", "disallowed")
}

func TestRecordHTTPActiveRequestsChange(t *testing.T) {
	meterProvider := noop.NewMeterProvider()
	metrics, err := NewMetrics(meterProvider, "test-service")
	if err != nil {
		t.Fatalf("failed to create metrics: %v", err)
	}

	ctx := context.Background()

	// Test increment
	metrics.RecordHTTPActiveRequestsChange(ctx, 1)

	// Test decrement
	metrics.RecordHTTPActiveRequestsChange(ctx, -1)
}

func TestExtractHost(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "normal URL",
			url:      "https://example.com/path",
			expected: "example.com",
		},
		{
			name:     "long URL",
			url:      "https://very-long-domain-name-that-exceeds-fifty-characters.com/very/long/path",
			expected: "very-long-domain-name-that-exceeds-fifty-characters.com",
		},
		{
			name:     "short URL",
			url:      "http://a.com",
			expected: "a.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractHost(tt.url)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGetGlobalMetrics(t *testing.T) {
	// This test just ensures the function doesn't panic
	metrics, err := GetGlobalMetrics("test-service")
	if err != nil {
		// With the noop provider from default otel, this should not error
		t.Errorf("expected no error, got %v", err)
	}
	if metrics == nil {
		t.Error("expected metrics to be non-nil")
	}
}

func BenchmarkRecordHTTPRequest(b *testing.B) {
	meterProvider := noop.NewMeterProvider()
	metrics, err := NewMetrics(meterProvider, "benchmark-service")
	if err != nil {
		b.Fatalf("failed to create metrics: %v", err)
	}

	ctx := context.Background()
	duration := 100 * time.Millisecond

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.RecordHTTPRequest(ctx, "GET", "/test", "200", duration)
	}
}

func BenchmarkRecordFetchOperation(b *testing.B) {
	meterProvider := noop.NewMeterProvider()
	metrics, err := NewMetrics(meterProvider, "benchmark-service")
	if err != nil {
		b.Fatalf("failed to create metrics: %v", err)
	}

	ctx := context.Background()
	duration := 500 * time.Millisecond

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.RecordFetchOperation(ctx, "https://example.com", "success", duration)
	}
}