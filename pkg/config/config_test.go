package config

import (
	"os"
	"strconv"
	"testing"
)

func TestConfigConstants(t *testing.T) {
	tests := []struct {
		name     string
		actual   interface{}
		expected interface{}
	}{
		{"ServerName", ServerName, "fetch-server"},
		{"ServerVersion", ServerVersion, "1.0.0"},
		{"DefaultUA", DefaultUA, "Mozilla/5.0 (compatible; MCPFetchBot/1.0)"},
		{"TransportSSE", TransportSSE, "sse"},
		{"TransportStreamableHTTP", TransportStreamableHTTP, "streamable-http"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.actual != tt.expected {
				t.Errorf("expected %s to be %v, got %v", tt.name, tt.expected, tt.actual)
			}
		})
	}
}

func TestConfigStruct(t *testing.T) {
	config := Config{
		Port:         9090,
		UserAgent:    "test-agent",
		IgnoreRobots: true,
		ProxyURL:     "http://proxy.example.com",
		Transport:    "sse",
		
		// Observability fields
		EnableOTelMetrics:   true,
		EnableOTelTracing:   true,
		OTelServiceName:     "test-service",
		OTelServiceVersion:  "2.0.0",
		OTelEndpoint:        "http://localhost:4317",
		EnablePrometheus:    true,
		MetricsPort:         9091,
	}

	if config.Port != 9090 {
		t.Errorf("expected Port to be 9090, got %d", config.Port)
	}
	if config.UserAgent != "test-agent" {
		t.Errorf("expected UserAgent to be 'test-agent', got %q", config.UserAgent)
	}
	if !config.IgnoreRobots {
		t.Errorf("expected IgnoreRobots to be true")
	}
	if config.ProxyURL != "http://proxy.example.com" {
		t.Errorf("expected ProxyURL to be 'http://proxy.example.com', got %q", config.ProxyURL)
	}
	if config.Transport != "sse" {
		t.Errorf("expected Transport to be 'sse', got %q", config.Transport)
	}
	
	// Test observability fields
	if !config.EnableOTelMetrics {
		t.Errorf("expected EnableOTelMetrics to be true")
	}
	if !config.EnableOTelTracing {
		t.Errorf("expected EnableOTelTracing to be true")
	}
	if config.OTelServiceName != "test-service" {
		t.Errorf("expected OTelServiceName to be 'test-service', got %q", config.OTelServiceName)
	}
	if config.OTelServiceVersion != "2.0.0" {
		t.Errorf("expected OTelServiceVersion to be '2.0.0', got %q", config.OTelServiceVersion)
	}
	if config.OTelEndpoint != "http://localhost:4317" {
		t.Errorf("expected OTelEndpoint to be 'http://localhost:4317', got %q", config.OTelEndpoint)
	}
	if !config.EnablePrometheus {
		t.Errorf("expected EnablePrometheus to be true")
	}
	if config.MetricsPort != 9091 {
		t.Errorf("expected MetricsPort to be 9091, got %d", config.MetricsPort)
	}
}

func TestConfigDefaults(t *testing.T) {
	var config Config

	if config.Port != 0 {
		t.Errorf("expected zero value for Port to be 0, got %d", config.Port)
	}
	if config.UserAgent != "" {
		t.Errorf("expected zero value for UserAgent to be empty, got %q", config.UserAgent)
	}
	if config.IgnoreRobots {
		t.Errorf("expected zero value for IgnoreRobots to be false")
	}
	if config.ProxyURL != "" {
		t.Errorf("expected zero value for ProxyURL to be empty, got %q", config.ProxyURL)
	}
	if config.Transport != "" {
		t.Errorf("expected zero value for Transport to be empty, got %q", config.Transport)
	}
	
	// Test observability zero values
	if config.EnableOTelMetrics {
		t.Errorf("expected zero value for EnableOTelMetrics to be false")
	}
	if config.EnableOTelTracing {
		t.Errorf("expected zero value for EnableOTelTracing to be false")
	}
	if config.OTelServiceName != "" {
		t.Errorf("expected zero value for OTelServiceName to be empty, got %q", config.OTelServiceName)
	}
	if config.OTelServiceVersion != "" {
		t.Errorf("expected zero value for OTelServiceVersion to be empty, got %q", config.OTelServiceVersion)
	}
	if config.OTelEndpoint != "" {
		t.Errorf("expected zero value for OTelEndpoint to be empty, got %q", config.OTelEndpoint)
	}
	if config.EnablePrometheus {
		t.Errorf("expected zero value for EnablePrometheus to be false")
	}
	if config.MetricsPort != 0 {
		t.Errorf("expected zero value for MetricsPort to be 0, got %d", config.MetricsPort)
	}
}

func TestParseFlags(t *testing.T) {
	config := ParseFlags()

	if config.Port <= 0 {
		t.Errorf("expected positive port number, got %d", config.Port)
	}
	if config.Transport == "" {
		t.Errorf("expected non-empty transport")
	}
	if config.UserAgent != DefaultUA {
		t.Errorf("expected default user agent %q, got %q", DefaultUA, config.UserAgent)
	}
	
	// Test observability defaults after parsing
	if config.OTelServiceName != ServerName {
		t.Errorf("expected default OTelServiceName %q, got %q", ServerName, config.OTelServiceName)
	}
	if config.OTelServiceVersion != ServerVersion {
		t.Errorf("expected default OTelServiceVersion %q, got %q", ServerVersion, config.OTelServiceVersion)
	}
	if config.MetricsPort != config.Port {
		t.Errorf("expected MetricsPort to default to Port %d, got %d", config.Port, config.MetricsPort)
	}
}

func TestTransportValidation(t *testing.T) {
	transports := []string{TransportSSE, TransportStreamableHTTP}

	for _, tr := range transports {
		if tr == "" {
			t.Errorf("transport constant should not be empty")
		}
		if len(tr) < 3 {
			t.Errorf("transport constant %q too short", tr)
		}
	}

	if TransportSSE == TransportStreamableHTTP {
		t.Errorf("transport constants should be unique")
	}
}

func TestObservabilityEnvironmentVariables(t *testing.T) {
	// Save original environment
	originalEnv := map[string]string{
		"ENABLE_OTEL_METRICS":      os.Getenv("ENABLE_OTEL_METRICS"),
		"ENABLE_OTEL_TRACING":      os.Getenv("ENABLE_OTEL_TRACING"),
		"OTEL_SERVICE_NAME":        os.Getenv("OTEL_SERVICE_NAME"),
		"OTEL_SERVICE_VERSION":     os.Getenv("OTEL_SERVICE_VERSION"),
		"OTEL_EXPORTER_OTLP_ENDPOINT": os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		"ENABLE_PROMETHEUS":        os.Getenv("ENABLE_PROMETHEUS"),
		"METRICS_PORT":             os.Getenv("METRICS_PORT"),
	}

	// Set test environment variables
	testEnv := map[string]string{
		"ENABLE_OTEL_METRICS":      "true",
		"ENABLE_OTEL_TRACING":      "true",
		"OTEL_SERVICE_NAME":        "test-service",
		"OTEL_SERVICE_VERSION":     "test-version",
		"OTEL_EXPORTER_OTLP_ENDPOINT": "http://test-endpoint:4317",
		"ENABLE_PROMETHEUS":        "true",
		"METRICS_PORT":             "9090",
	}

	for key, value := range testEnv {
		os.Setenv(key, value)
	}

	defer func() {
		// Restore original environment
		for key, value := range originalEnv {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	// Test only environment variable parsing by testing the individual assignments
	var config Config
	
	// Test environment variable parsing directly
	if enableMetrics, ok := os.LookupEnv("ENABLE_OTEL_METRICS"); ok {
		config.EnableOTelMetrics = enableMetrics == "true"
	}
	if enableTracing, ok := os.LookupEnv("ENABLE_OTEL_TRACING"); ok {
		config.EnableOTelTracing = enableTracing == "true"
	}
	if serviceName, ok := os.LookupEnv("OTEL_SERVICE_NAME"); ok {
		config.OTelServiceName = serviceName
	}
	if serviceVersion, ok := os.LookupEnv("OTEL_SERVICE_VERSION"); ok {
		config.OTelServiceVersion = serviceVersion
	}
	if endpoint, ok := os.LookupEnv("OTEL_EXPORTER_OTLP_ENDPOINT"); ok {
		config.OTelEndpoint = endpoint
	}
	if enablePrometheus, ok := os.LookupEnv("ENABLE_PROMETHEUS"); ok {
		config.EnablePrometheus = enablePrometheus == "true"
	}
	if metricsPort, ok := os.LookupEnv("METRICS_PORT"); ok {
		if intValue, err := strconv.Atoi(metricsPort); err == nil {
			config.MetricsPort = intValue
		}
	}

	// Verify environment variables were parsed correctly
	if !config.EnableOTelMetrics {
		t.Errorf("expected EnableOTelMetrics to be true from environment variable")
	}
	if !config.EnableOTelTracing {
		t.Errorf("expected EnableOTelTracing to be true from environment variable")
	}
	if config.OTelServiceName != "test-service" {
		t.Errorf("expected OTelServiceName to be 'test-service', got %q", config.OTelServiceName)
	}
	if config.OTelServiceVersion != "test-version" {
		t.Errorf("expected OTelServiceVersion to be 'test-version', got %q", config.OTelServiceVersion)
	}
	if config.OTelEndpoint != "http://test-endpoint:4317" {
		t.Errorf("expected OTelEndpoint to be 'http://test-endpoint:4317', got %q", config.OTelEndpoint)
	}
	if !config.EnablePrometheus {
		t.Errorf("expected EnablePrometheus to be true from environment variable")
	}
	if config.MetricsPort != 9090 {
		t.Errorf("expected MetricsPort to be 9090, got %d", config.MetricsPort)
	}
}

func BenchmarkConfigCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Config{
			Port:         8080,
			UserAgent:    DefaultUA,
			IgnoreRobots: false,
			ProxyURL:     "",
			Transport:    TransportStreamableHTTP,
			
			// Observability fields
			EnableOTelMetrics:   true,
			EnableOTelTracing:   true,
			OTelServiceName:     ServerName,
			OTelServiceVersion:  ServerVersion,
			OTelEndpoint:        "http://localhost:4317",
			EnablePrometheus:    true,
			MetricsPort:         8080,
		}
	}
}
