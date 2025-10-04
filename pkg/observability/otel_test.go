package observability

import (
	"context"
	"testing"

	"github.com/stackloklabs/gofetch/pkg/config"
)

func TestNewObservabilityConfig(t *testing.T) {
	cfg := config.Config{
		EnableOTelMetrics:  true,
		EnableOTelTracing:  true,
		OTelServiceName:    "test-service",
		OTelServiceVersion: "1.0.0",
		OTelEndpoint:       "http://localhost:4317",
	}

	obsConfig := NewObservabilityConfig(cfg)

	if obsConfig.ServiceName != "test-service" {
		t.Errorf("expected ServiceName 'test-service', got %q", obsConfig.ServiceName)
	}
	if obsConfig.ServiceVersion != "1.0.0" {
		t.Errorf("expected ServiceVersion '1.0.0', got %q", obsConfig.ServiceVersion)
	}
	if obsConfig.Endpoint != "http://localhost:4317" {
		t.Errorf("expected Endpoint 'http://localhost:4317', got %q", obsConfig.Endpoint)
	}
	if !obsConfig.EnableMetrics {
		t.Errorf("expected EnableMetrics to be true")
	}
	if !obsConfig.EnableTracing {
		t.Errorf("expected EnableTracing to be true")
	}
}

func TestSetupWithDisabledObservability(t *testing.T) {
	obsConfig := ObservabilityConfig{
		ServiceName:      "test-service",
		ServiceVersion:   "1.0.0",
		EnableMetrics:    false,
		EnableTracing:    false,
	}

	ctx := context.Background()
	tel, err := Setup(ctx, obsConfig)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if tel == nil {
		t.Fatal("expected telemetry instance to be created")
	}

	// When all observability features are disabled, we should get empty telemetry
	if tel.TracerProvider != nil {
		t.Errorf("expected nil TracerProvider when tracing is disabled")
	}
	if tel.MeterProvider != nil {
		t.Errorf("expected nil MeterProvider when metrics are disabled")
	}
}


func TestTelemetryShutdown(t *testing.T) {
	tel := &Telemetry{}

	// Test shutdown with no shutdown functions
	ctx := context.Background()
	err := tel.Shutdown(ctx)
	if err != nil {
		t.Errorf("expected no error on shutdown with no functions, got %v", err)
	}

	// Test shutdown with a mock shutdown function
	called := false
	tel.shutdownFuncs = []func(context.Context) error{
		func(ctx context.Context) error {
			called = true
			return nil
		},
	}

	err = tel.Shutdown(ctx)
	if err != nil {
		t.Errorf("expected no error on shutdown, got %v", err)
	}
	if !called {
		t.Errorf("expected shutdown function to be called")
	}
}

func BenchmarkNewObservabilityConfig(b *testing.B) {
	cfg := config.Config{
		EnableOTelMetrics:  true,
		EnableOTelTracing:  true,
		OTelServiceName:    "benchmark-service",
		OTelServiceVersion: "1.0.0",
		OTelEndpoint:       "http://localhost:4317",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewObservabilityConfig(cfg)
	}
}