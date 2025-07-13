package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stackloklabs/gofetch/pkg/config"
)

func TestNewFetchServer(t *testing.T) {
	cfg := config.Config{
		Port:         8080,
		UserAgent:    "test-agent",
		IgnoreRobots: true,
		Transport:    config.TransportSSE,
	}

	server := NewFetchServer(cfg)

	if server == nil {
		t.Fatal("expected server to be created")
	}
	if server.config.Port != 8080 {
		t.Errorf("expected port 8080, got %d", server.config.Port)
	}
	if server.config.UserAgent != "test-agent" {
		t.Errorf("expected user agent 'test-agent', got %q", server.config.UserAgent)
	}
	if !server.config.IgnoreRobots {
		t.Errorf("expected IgnoreRobots to be true")
	}
	if server.fetcher == nil {
		t.Error("expected fetcher to be initialized")
	}
	if server.mcpServer == nil {
		t.Error("expected MCP server to be initialized")
	}
}

func TestNewFetchServerWithProxy(t *testing.T) {
	cfg := config.Config{
		Port:      8080,
		UserAgent: "test-agent",
		ProxyURL:  "http://proxy.example.com:8080",
		Transport: config.TransportSSE,
	}

	server := NewFetchServer(cfg)

	if server == nil {
		t.Fatal("expected server to be created")
	}
	if server.config.ProxyURL != "http://proxy.example.com:8080" {
		t.Errorf("expected proxy URL to be set, got %q", server.config.ProxyURL)
	}
}

func TestFetchParams(t *testing.T) {
	maxLength := 1000
	startIndex := 100

	params := FetchParams{
		URL:        "https://example.com",
		MaxLength:  &maxLength,
		StartIndex: &startIndex,
		Raw:        true,
	}

	if params.URL != "https://example.com" {
		t.Errorf("expected URL 'https://example.com', got %q", params.URL)
	}
	if params.MaxLength == nil || *params.MaxLength != 1000 {
		t.Errorf("expected MaxLength 1000, got %v", params.MaxLength)
	}
	if params.StartIndex == nil || *params.StartIndex != 100 {
		t.Errorf("expected StartIndex 100, got %v", params.StartIndex)
	}
	if !params.Raw {
		t.Errorf("expected Raw to be true")
	}
}

func TestHandleFetchTool(t *testing.T) {
	cfg := config.Config{
		Port:      8080,
		UserAgent: "test-agent",
		Transport: config.TransportSSE,
	}

	server := NewFetchServer(cfg)

	// Create a test server to serve content
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html><body><h1>Test Content</h1></body></html>"))
	}))
	defer testServer.Close()

	ctx := context.Background()
	params := &mcp.CallToolParamsFor[FetchParams]{
		Name: "fetch",
		Arguments: FetchParams{
			URL: testServer.URL,
			Raw: false,
		},
	}

	result, err := server.handleFetchTool(ctx, nil, params)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected result to be returned")
	}
	if len(result.Content) == 0 {
		t.Error("expected content to be returned")
	}
}

func TestHandleFetchToolWithParams(t *testing.T) {
	cfg := config.Config{
		Port:      8080,
		UserAgent: "test-agent",
		Transport: config.TransportSSE,
	}

	server := NewFetchServer(cfg)

	// Create a test server
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html><body><p>Long content here</p></body></html>"))
	}))
	defer testServer.Close()

	maxLength := 50
	startIndex := 0

	ctx := context.Background()
	params := &mcp.CallToolParamsFor[FetchParams]{
		Name: "fetch",
		Arguments: FetchParams{
			URL:        testServer.URL,
			MaxLength:  &maxLength,
			StartIndex: &startIndex,
			Raw:        true,
		},
	}

	result, err := server.handleFetchTool(ctx, nil, params)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected result to be returned")
	}
	if len(result.Content) == 0 {
		t.Error("expected content to be returned")
	}
}

func TestHandleFetchToolError(t *testing.T) {
	cfg := config.Config{
		Port:      8080,
		UserAgent: "test-agent",
		Transport: config.TransportSSE,
	}

	server := NewFetchServer(cfg)

	ctx := context.Background()
	params := &mcp.CallToolParamsFor[FetchParams]{
		Name: "fetch",
		Arguments: FetchParams{
			URL: "http://invalid-url-that-does-not-exist.invalid",
		},
	}

	result, err := server.handleFetchTool(ctx, nil, params)

	if err == nil {
		t.Error("expected error for invalid URL")
	}
	if result != nil {
		t.Error("expected no result on error")
	}
}

func TestStartUnsupportedTransport(t *testing.T) {
	cfg := config.Config{
		Port:      8080,
		Transport: "invalid-transport",
	}

	server := NewFetchServer(cfg)
	err := server.Start()

	if err == nil {
		t.Error("expected error for unsupported transport")
	}
	if err.Error() != "unsupported transport type: invalid-transport" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestLogServerStartup(t *testing.T) {
	cfg := config.Config{
		Port:         9090,
		UserAgent:    "test-agent",
		IgnoreRobots: true,
		ProxyURL:     "http://proxy.example.com",
		Transport:    config.TransportSSE,
	}

	server := NewFetchServer(cfg)

	// This test just ensures logServerStartup doesn't panic
	// In a real test environment, you might want to capture logs
	server.logServerStartup()
}

func TestLogServerStartupStreamableHTTP(t *testing.T) {
	cfg := config.Config{
		Port:      8080,
		Transport: config.TransportStreamableHTTP,
	}

	server := NewFetchServer(cfg)
	server.logServerStartup()
}

func TestConfigFieldsPreserved(t *testing.T) {
	cfg := config.Config{
		Port:         9999,
		UserAgent:    "custom-agent",
		IgnoreRobots: true,
		ProxyURL:     "http://custom-proxy.com",
		Transport:    config.TransportStreamableHTTP,
	}

	server := NewFetchServer(cfg)

	if server.config.Port != cfg.Port {
		t.Errorf("expected port %d, got %d", cfg.Port, server.config.Port)
	}
	if server.config.UserAgent != cfg.UserAgent {
		t.Errorf("expected user agent %q, got %q", cfg.UserAgent, server.config.UserAgent)
	}
	if server.config.IgnoreRobots != cfg.IgnoreRobots {
		t.Errorf("expected IgnoreRobots %v, got %v", cfg.IgnoreRobots, server.config.IgnoreRobots)
	}
	if server.config.ProxyURL != cfg.ProxyURL {
		t.Errorf("expected proxy URL %q, got %q", cfg.ProxyURL, server.config.ProxyURL)
	}
	if server.config.Transport != cfg.Transport {
		t.Errorf("expected transport %q, got %q", cfg.Transport, server.config.Transport)
	}
}

func BenchmarkNewFetchServer(b *testing.B) {
	cfg := config.Config{
		Port:      8080,
		UserAgent: "benchmark-agent",
		Transport: config.TransportSSE,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewFetchServer(cfg)
	}
}

func BenchmarkHandleFetchTool(b *testing.B) {
	cfg := config.Config{
		Port:      8080,
		UserAgent: "benchmark-agent",
		Transport: config.TransportSSE,
	}

	server := NewFetchServer(cfg)

	// Create a test server
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html><body><h1>Benchmark Content</h1></body></html>"))
	}))
	defer testServer.Close()

	ctx := context.Background()
	params := &mcp.CallToolParamsFor[FetchParams]{
		Name: "fetch",
		Arguments: FetchParams{
			URL: testServer.URL,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = server.handleFetchTool(ctx, nil, params)
	}
}
