package observability

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel/exporters/prometheus"
)

func TestTelemetryPrometheusHandler(t *testing.T) {
	// Test with nil PrometheusExporter
	tel := &Telemetry{}
	handler := tel.PrometheusHandler()
	
	if handler == nil {
		t.Fatal("expected handler to be non-nil")
	}

	// Test the handler with a request
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestTelemetryPrometheusHandlerWithExporter(t *testing.T) {
	// Create a Prometheus exporter
	exporter, err := prometheus.New()
	if err != nil {
		t.Fatalf("failed to create Prometheus exporter: %v", err)
	}

	tel := &Telemetry{
		PrometheusExporter: exporter,
	}
	
	handler := tel.PrometheusHandler()
	if handler == nil {
		t.Fatal("expected handler to be non-nil")
	}

	// Test the handler with a request
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Check that we get some metrics output
	body := w.Body.String()
	if len(body) == 0 {
		t.Error("expected metrics output to be non-empty")
	}
}

func TestGetPrometheusHandler(t *testing.T) {
	handler, err := GetPrometheusHandler()
	
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if handler == nil {
		t.Fatal("expected handler to be non-nil")
	}

	// Test the handler with a request
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestPrometheusHandlerContentType(t *testing.T) {
	handler, err := GetPrometheusHandler()
	if err != nil {
		t.Fatalf("failed to get Prometheus handler: %v", err)
	}

	// Test with Accept header for Prometheus format
	req := httptest.NewRequest("GET", "/metrics", nil)
	req.Header.Set("Accept", "application/openmetrics-text; version=1.0.0, text/plain")
	w := httptest.NewRecorder()
	
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Check content type
	contentType := w.Header().Get("Content-Type")
	if contentType == "" {
		t.Error("expected Content-Type header to be set")
	}
}

func BenchmarkTelemetryPrometheusHandler(b *testing.B) {
	tel := &Telemetry{}
	handler := tel.PrometheusHandler()
	req := httptest.NewRequest("GET", "/metrics", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}

func BenchmarkGetPrometheusHandler(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = GetPrometheusHandler()
	}
}