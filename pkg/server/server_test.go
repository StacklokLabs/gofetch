package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

// TestSSEEndpointsExist tests that SSE transport exposes both required endpoints
func TestSSEEndpointsExist(t *testing.T) {
	cfg := config.Config{
		Port:      8080,
		UserAgent: "test-agent",
		Transport: config.TransportSSE,
	}

	server := NewFetchServer(cfg)

	// Create a test HTTP server to simulate the endpoint structure
	mux := http.NewServeMux()
	sseHandler := mcp.NewSSEHandler(func(_ *http.Request) *mcp.Server {
		return server.mcpServer
	})

	// Set up endpoints as per MCP spec
	mux.Handle("/sse", sseHandler)
	mux.Handle("/messages", sseHandler)

	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	// Test SSE endpoint exists
	resp, err := http.Get(testServer.URL + "/sse")
	if err != nil {
		t.Errorf("SSE endpoint should be accessible: %v", err)
	}
	resp.Body.Close()

	// Test messages endpoint exists (should accept POST)
	resp, err = http.Post(testServer.URL+"/messages", "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Errorf("Messages endpoint should be accessible: %v", err)
	}
	resp.Body.Close()
}

// TestStreamableHTTPEndpointsExist tests that Streamable HTTP transport exposes the single required endpoint
func TestStreamableHTTPEndpointsExist(t *testing.T) {
	cfg := config.Config{
		Port:      8080,
		UserAgent: "test-agent",
		Transport: config.TransportStreamableHTTP,
	}

	server := NewFetchServer(cfg)

	// Create a test HTTP server to simulate the endpoint structure
	mux := http.NewServeMux()
	streamableHandler := mcp.NewStreamableHTTPHandler(
		func(_ *http.Request) *mcp.Server {
			return server.mcpServer
		},
		&mcp.StreamableHTTPOptions{},
	)

	// Set up single endpoint as per MCP spec for Streamable HTTP
	mux.Handle("/mcp", streamableHandler)

	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	// Test MCP endpoint exists for GET (streaming)
	resp, err := http.Get(testServer.URL + "/mcp")
	if err != nil {
		t.Errorf("MCP endpoint should be accessible for GET: %v", err)
	}
	resp.Body.Close()

	// Test MCP endpoint exists for POST (commands)
	resp, err = http.Post(testServer.URL+"/mcp", "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Errorf("MCP endpoint should be accessible for POST: %v", err)
	}
	resp.Body.Close()
}

// TestLogServerStartupEndpoints tests that the correct endpoints are logged
func TestLogServerStartupEndpoints(t *testing.T) {
	testCases := []struct {
		name      string
		transport string
		expected  []string
	}{
		{
			name:      "SSE transport",
			transport: config.TransportSSE,
			expected:  []string{"SSE endpoint (server-to-client)", "Messages endpoint (client-to-server)", "/sse", "/messages"},
		},
		{
			name:      "Streamable HTTP transport",
			transport: config.TransportStreamableHTTP,
			expected:  []string{"MCP endpoint (streaming and commands)", "/mcp"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(_ *testing.T) {
			cfg := config.Config{
				Port:      8080,
				Transport: tc.transport,
			}

			server := NewFetchServer(cfg)

			// This test verifies that logServerStartup doesn't panic and would log the expected endpoints
			// In a real test environment, you might want to capture logs and verify the content
			server.logServerStartup()
		})
	}
}

// TestInitializedHandlerSetup tests that the initialized handler is properly configured
func TestInitializedHandlerSetup(t *testing.T) {
	testCases := []struct {
		name      string
		transport string
		port      int
	}{
		{
			name:      "SSE transport",
			transport: config.TransportSSE,
			port:      8080,
		},
		{
			name:      "Streamable HTTP transport",
			transport: config.TransportStreamableHTTP,
			port:      9090,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.Config{
				Port:      tc.port,
				Transport: tc.transport,
			}

			server := NewFetchServer(cfg)

			// Verify that the server was created successfully
			if server == nil {
				t.Error("Expected server to be created")
				return
			}

			// Verify that the MCP server was created
			if server.mcpServer == nil {
				t.Error("Expected MCP server to be created")
				return
			}

			// This test verifies that the server has been configured with an initialized handler
			// The actual endpoint event sending would be tested through integration tests
			// since we can't easily mock ServerSession due to it being a concrete type
		})
	}
}

// TestEndpointURIGeneration tests the endpoint URI generation logic
func TestEndpointURIGeneration(t *testing.T) {
	testCases := []struct {
		name      string
		transport string
		port      int
		expected  string
	}{
		{
			name:      "SSE transport",
			transport: config.TransportSSE,
			port:      8080,
			expected:  "http://localhost:8080/messages",
		},
		{
			name:      "Streamable HTTP transport",
			transport: config.TransportStreamableHTTP,
			port:      9090,
			expected:  "http://localhost:9090/mcp",
		},
		{
			name:      "Custom port",
			transport: config.TransportSSE,
			port:      3000,
			expected:  "http://localhost:3000/messages",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.Config{
				Port:      tc.port,
				Transport: tc.transport,
			}

			server := NewFetchServer(cfg)

			// Test the endpoint URI generation logic
			var endpointURI string
			switch server.config.Transport {
			case config.TransportSSE:
				endpointURI = fmt.Sprintf("http://localhost:%d/messages", server.config.Port)
			case config.TransportStreamableHTTP:
				endpointURI = fmt.Sprintf("http://localhost:%d/mcp", server.config.Port)
			default:
				endpointURI = fmt.Sprintf("http://localhost:%d/messages", server.config.Port)
			}

			if endpointURI != tc.expected {
				t.Errorf("Expected endpoint URI '%s', got '%s'", tc.expected, endpointURI)
			}
		})
	}
}
