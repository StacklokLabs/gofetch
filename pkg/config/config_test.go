package config

import "testing"

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

func BenchmarkConfigCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Config{
			Port:         8080,
			UserAgent:    DefaultUA,
			IgnoreRobots: false,
			ProxyURL:     "",
			Transport:    TransportStreamableHTTP,
		}
	}
}