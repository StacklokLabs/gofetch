package server

import (
	"context"
	"encoding/json"
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
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html><body><h1>Test Content</h1></body></html>"))
	}))
	defer testServer.Close()

	ctx := context.Background()
	params := FetchParams{
		URL: testServer.URL,
		Raw: false,
	}

	result, _, err := server.handleFetchTool(ctx, nil, params)

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
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html><body><p>Long content here</p></body></html>"))
	}))
	defer testServer.Close()

	maxLength := 50
	startIndex := 0

	ctx := context.Background()
	params := FetchParams{
		URL:        testServer.URL,
		MaxLength:  &maxLength,
		StartIndex: &startIndex,
		Raw:        true,
	}

	result, _, err := server.handleFetchTool(ctx, nil, params)

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
	params := FetchParams{
		URL: "http://invalid-url-that-does-not-exist.invalid",
	}

	result, _, err := server.handleFetchTool(ctx, nil, params)

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

func TestLogServerStartup(_ *testing.T) {
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

func TestLogServerStartupStreamableHTTP(_ *testing.T) {
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
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html><body><h1>Benchmark Content</h1></body></html>"))
	}))
	defer testServer.Close()

	ctx := context.Background()
	params := FetchParams{
		URL: testServer.URL,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = server.handleFetchTool(ctx, nil, params)
	}
}

func TestFetchToolDefinition(t *testing.T) {
	t.Parallel()

	fs := NewFetchServer(config.Config{Transport: config.TransportStreamableHTTP})

	ctx := context.Background()
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serverSession, err := fs.mcpServer.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	defer serverSession.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.0.1"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer clientSession.Close()

	res, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(res.Tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(res.Tools))
	}
	tool := res.Tools[0]

	if tool.Name != "fetch" {
		t.Errorf("expected tool name 'fetch', got %q", tool.Name)
	}
	if tool.Title == "" {
		t.Error("expected tool title to be set")
	}

	a := tool.Annotations
	if a == nil {
		t.Fatal("expected tool annotations to be set")
	}
	if !a.ReadOnlyHint {
		t.Error("expected readOnlyHint to be true")
	}
	if !a.IdempotentHint {
		t.Error("expected idempotentHint to be true")
	}
	if a.OpenWorldHint == nil || !*a.OpenWorldHint {
		t.Error("expected openWorldHint to be true")
	}

	// Round-trip the schema through JSON so the test checks what clients see.
	raw, err := json.Marshal(tool.InputSchema)
	if err != nil {
		t.Fatalf("marshal input schema: %v", err)
	}
	var schema struct {
		Required   []string `json:"required"`
		Properties map[string]struct {
			Description string   `json:"description"`
			Minimum     *float64 `json:"minimum"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatalf("unmarshal input schema: %v", err)
	}

	for _, name := range []string{"url", "max_length", "start_index", "raw"} {
		prop, ok := schema.Properties[name]
		if !ok {
			t.Errorf("expected input schema property %q", name)
			continue
		}
		if prop.Description == "" {
			t.Errorf("expected description for input schema property %q", name)
		}
	}
	if len(schema.Required) != 1 || schema.Required[0] != "url" {
		t.Errorf("expected only 'url' to be required, got %v", schema.Required)
	}
	if m := schema.Properties["max_length"].Minimum; m == nil || *m != 1 {
		t.Errorf("expected max_length minimum of 1, got %v", m)
	}
	if m := schema.Properties["start_index"].Minimum; m == nil || *m != 0 {
		t.Errorf("expected start_index minimum of 0, got %v", m)
	}
}
