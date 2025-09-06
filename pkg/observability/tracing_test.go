package observability

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestNewTraceHelper(t *testing.T) {
	serviceName := "test-service"
	helper := NewTraceHelper(serviceName)

	if helper == nil {
		t.Fatal("expected trace helper to be created")
	}
	if helper.tracer == nil {
		t.Error("expected tracer to be initialized")
	}
}

func TestStartHTTPSpan(t *testing.T) {
	// Set up a noop tracer provider
	otel.SetTracerProvider(noop.NewTracerProvider())

	helper := NewTraceHelper("test-service")
	ctx := context.Background()

	// Create a test HTTP request
	req, err := http.NewRequest("GET", "https://example.com/test", nil)
	if err != nil {
		t.Fatalf("failed to create test request: %v", err)
	}
	req.Host = "example.com"
	req.Header.Set("User-Agent", "test-agent")

	newCtx, span := helper.StartHTTPSpan(ctx, req)

	if newCtx == nil {
		t.Error("expected context to be returned")
	}
	if span == nil {
		t.Error("expected span to be returned")
	}

	// The span should be valid (though it's a noop)
	if !span.IsRecording() {
		// With noop tracer, spans won't be recording
		t.Log("noop span is not recording (expected)")
	}

	span.End()
}

func TestFinishHTTPSpan(t *testing.T) {
	otel.SetTracerProvider(noop.NewTracerProvider())
	helper := NewTraceHelper("test-service")

	req, _ := http.NewRequest("GET", "https://example.com/test", nil)
	_, span := helper.StartHTTPSpan(context.Background(), req)

	// Test successful response
	helper.FinishHTTPSpan(span, 200, nil)

	// Create another span for error test
	_, errorSpan := helper.StartHTTPSpan(context.Background(), req)
	// Test error response
	helper.FinishHTTPSpan(errorSpan, 500, http.ErrServerClosed)
}

func TestStartFetchSpan(t *testing.T) {
	otel.SetTracerProvider(noop.NewTracerProvider())
	helper := NewTraceHelper("test-service")

	ctx := context.Background()
	fetchURL := "https://example.com/api/data"

	newCtx, span := helper.StartFetchSpan(ctx, fetchURL)

	if newCtx == nil {
		t.Error("expected context to be returned")
	}
	if span == nil {
		t.Error("expected span to be returned")
	}

	span.End()
}

func TestFinishFetchSpan(t *testing.T) {
	otel.SetTracerProvider(noop.NewTracerProvider())
	helper := NewTraceHelper("test-service")

	ctx := context.Background()
	_, span := helper.StartFetchSpan(ctx, "https://example.com")

	// Test successful fetch
	helper.FinishFetchSpan(span, 200, 1024, nil)

	// Create another span for error test
	_, errorSpan := helper.StartFetchSpan(ctx, "https://example.com")
	// Test failed fetch
	helper.FinishFetchSpan(errorSpan, 404, 0, errors.New("not found"))
}

func TestStartProcessContentSpan(t *testing.T) {
	otel.SetTracerProvider(noop.NewTracerProvider())
	helper := NewTraceHelper("test-service")

	ctx := context.Background()
	contentType := "text/html"

	newCtx, span := helper.StartProcessContentSpan(ctx, contentType)

	if newCtx == nil {
		t.Error("expected context to be returned")
	}
	if span == nil {
		t.Error("expected span to be returned")
	}

	span.End()
}

func TestFinishProcessContentSpan(t *testing.T) {
	otel.SetTracerProvider(noop.NewTracerProvider())
	helper := NewTraceHelper("test-service")

	ctx := context.Background()
	_, span := helper.StartProcessContentSpan(ctx, "text/html")

	// Test successful content processing
	helper.FinishProcessContentSpan(span, 2048, 1024, nil)

	// Create another span for error test
	_, errorSpan := helper.StartProcessContentSpan(ctx, "text/html")
	// Test failed processing
	helper.FinishProcessContentSpan(errorSpan, 2048, 0, http.ErrServerClosed)
}

func TestStartRobotsCheckSpan(t *testing.T) {
	otel.SetTracerProvider(noop.NewTracerProvider())
	helper := NewTraceHelper("test-service")

	ctx := context.Background()
	checkURL := "https://example.com/robots.txt"

	newCtx, span := helper.StartRobotsCheckSpan(ctx, checkURL)

	if newCtx == nil {
		t.Error("expected context to be returned")
	}
	if span == nil {
		t.Error("expected span to be returned")
	}

	span.End()
}

func TestFinishRobotsCheckSpan(t *testing.T) {
	otel.SetTracerProvider(noop.NewTracerProvider())
	helper := NewTraceHelper("test-service")

	ctx := context.Background()
	_, span := helper.StartRobotsCheckSpan(ctx, "https://example.com")

	// Test allowed result
	helper.FinishRobotsCheckSpan(span, true, nil)

	// Create another span for error test
	_, errorSpan := helper.StartRobotsCheckSpan(ctx, "https://example.com")
	// Test disallowed with error
	helper.FinishRobotsCheckSpan(errorSpan, false, http.ErrServerClosed)
}

func TestAddSpanEvent(t *testing.T) {
	otel.SetTracerProvider(noop.NewTracerProvider())
	helper := NewTraceHelper("test-service")

	ctx := context.Background()

	// Test with no active span (should not panic)
	helper.AddSpanEvent(ctx, "test-event")

	// Test with active span
	req, _ := http.NewRequest("GET", "https://example.com", nil)
	spanCtx, span := helper.StartHTTPSpan(ctx, req)
	helper.AddSpanEvent(spanCtx, "test-event-with-span")
	span.End()
}

func TestExtractHostFromURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "valid HTTP URL",
			url:      "http://example.com/path",
			expected: "example.com",
		},
		{
			name:     "valid HTTPS URL",
			url:      "https://example.com:8080/path",
			expected: "example.com:8080",
		},
		{
			name:     "invalid URL",
			url:      "not-a-url",
			expected: "unknown",
		},
		{
			name:     "URL with subdomain",
			url:      "https://api.example.com/v1/users",
			expected: "api.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractHostFromURL(tt.url)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGetTraceHelper(t *testing.T) {
	helper := GetTraceHelper("test-service")

	if helper == nil {
		t.Fatal("expected trace helper to be created")
	}
	if helper.tracer == nil {
		t.Error("expected tracer to be initialized")
	}
}

func BenchmarkStartHTTPSpan(b *testing.B) {
	otel.SetTracerProvider(noop.NewTracerProvider())
	helper := NewTraceHelper("benchmark-service")
	ctx := context.Background()

	req, _ := http.NewRequest("GET", "https://example.com/test", nil)
	req.Host = "example.com"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, span := helper.StartHTTPSpan(ctx, req)
		span.End()
	}
}

func BenchmarkStartFetchSpan(b *testing.B) {
	otel.SetTracerProvider(noop.NewTracerProvider())
	helper := NewTraceHelper("benchmark-service")
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, span := helper.StartFetchSpan(ctx, "https://example.com")
		span.End()
	}
}