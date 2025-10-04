// Package server provides the MCP server implementation for gofetching web content.
package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stackloklabs/gofetch/pkg/config"
	"github.com/stackloklabs/gofetch/pkg/fetcher"
	"github.com/stackloklabs/gofetch/pkg/observability"
	"github.com/stackloklabs/gofetch/pkg/processor"
	"github.com/stackloklabs/gofetch/pkg/robots"
)

// FetchParams defines the input parameters for the fetch tool
type FetchParams struct {
	URL        string `json:"url" mcp:"URL to fetch"`
	MaxLength  *int   `json:"max_length,omitempty" mcp:"Maximum number of characters to return"`
	StartIndex *int   `json:"start_index,omitempty" mcp:"Start index for truncated content"`
	Raw        bool   `json:"raw,omitempty" mcp:"Get the actual HTML content without simplification"`
}

// FetchServer represents the MCP server for fetching web content
type FetchServer struct {
	config      config.Config
	fetcher     *fetcher.HTTPFetcher
	mcpServer   *mcp.Server
	telemetry   *observability.Telemetry
	metrics     *observability.Metrics
	traceHelper *observability.TraceHelper
}

// NewFetchServer creates a new fetch server instance
func NewFetchServer(cfg config.Config) *FetchServer {
	// Initialize observability
	var telemetry *observability.Telemetry
	var metrics *observability.Metrics
	var traceHelper *observability.TraceHelper
	
	if cfg.EnableOTelMetrics || cfg.EnableOTelTracing || cfg.EnablePrometheus {
		ctx := context.Background()
		obsConfig := observability.NewObservabilityConfig(cfg)
		
		var err error
		telemetry, err = observability.Setup(ctx, obsConfig)
		if err != nil {
			log.Printf("Failed to setup observability: %v", err)
		} else {
			// Initialize metrics and tracing helpers
			if telemetry.MeterProvider != nil {
				metrics, err = observability.NewMetrics(telemetry.MeterProvider, obsConfig.ServiceName)
				if err != nil {
					log.Printf("Failed to initialize metrics: %v", err)
				}
			}
			
			if telemetry.TracerProvider != nil {
				traceHelper = observability.NewTraceHelper(obsConfig.ServiceName)
			}
		}
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	
	// Add OpenTelemetry HTTP instrumentation if tracing is enabled
	if traceHelper != nil {
		client.Transport = otelhttp.NewTransport(client.Transport)
	}

	// Configure proxy if provided
	if cfg.ProxyURL != "" {
		if proxyURLParsed, err := url.Parse(cfg.ProxyURL); err == nil {
			if traceHelper != nil {
				// Wrap the proxy transport with OpenTelemetry
				client.Transport = otelhttp.NewTransport(&http.Transport{
					Proxy: http.ProxyURL(proxyURLParsed),
				})
			} else {
				client.Transport = &http.Transport{
					Proxy: http.ProxyURL(proxyURLParsed),
				}
			}
		}
	}

	// Create components
	robotsChecker := robots.NewChecker(cfg.UserAgent, cfg.IgnoreRobots, client)
	contentProcessor := processor.NewContentProcessor()
	httpFetcher := fetcher.NewHTTPFetcher(client, robotsChecker, contentProcessor, cfg.UserAgent, metrics)

	fs := &FetchServer{
		config:      cfg,
		fetcher:     httpFetcher,
		telemetry:   telemetry,
		metrics:     metrics,
		traceHelper: traceHelper,
	}

	// Create MCP server with proper implementation details
	// Capabilities are automatically generated based on registered tools/resources
	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    config.ServerName,
		Version: config.ServerVersion,
	}, &mcp.ServerOptions{
		InitializedHandler: fs.handleInitialized,
	})

	fs.mcpServer = mcpServer

	// Setup tools
	fs.setupTools()

	return fs
}

// handleInitialized sends an endpoint event to the client after initialization
func (fs *FetchServer) handleInitialized(ctx context.Context, initRequest *mcp.InitializedRequest) {
	// Record MCP session metrics
	if fs.metrics != nil {
		fs.metrics.RecordMCPSessionChange(ctx, 1)
		// Note: We'd need to hook into session close to decrement this
	}
	
	// Build the endpoint URI based on the current server configuration
	var endpointURI string
	switch fs.config.Transport {
	case config.TransportSSE:
		endpointURI = fmt.Sprintf("http://localhost:%d/messages", fs.config.Port)
	case config.TransportStreamableHTTP:
		endpointURI = fmt.Sprintf("http://localhost:%d/mcp", fs.config.Port)
	default:
		endpointURI = fmt.Sprintf("http://localhost:%d/messages", fs.config.Port)
	}

	// Send endpoint event as a log message with structured data
	err := initRequest.Session.Log(ctx, &mcp.LoggingMessageParams{
		Level: "info",
		Data: map[string]interface{}{
			"type":         "endpoint_event",
			"message":      "Client must use this endpoint for sending messages",
			"endpoint_uri": endpointURI,
		},
		Logger: "gofetch-server",
	})

	if err != nil {
		log.Printf("Failed to send endpoint event: %v", err)
	} else {
		log.Printf("Sent endpoint event to client %s: %s", initRequest.Session.ID(), endpointURI)
	}
}

// setupTools registers the fetch tool with the MCP server
func (fs *FetchServer) setupTools() {
	fetchTool := &mcp.Tool{
		Name:        "fetch",
		Description: "Fetches a URL from the internet and optionally extracts its contents as markdown.",
	}

	mcp.AddTool(fs.mcpServer, fetchTool, fs.handleFetchTool)
}

// handleFetchTool processes fetch tool requests
func (fs *FetchServer) handleFetchTool(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	params FetchParams,
) (*mcp.CallToolResult, any, error) {
	log.Printf("Tool call received: fetch")

	// Start tracing span if enabled
	if fs.traceHelper != nil {
		var span trace.Span
		ctx, span = fs.traceHelper.StartFetchSpan(ctx, params.URL)
		defer span.End()
	}

	// Record start time for metrics
	startTime := time.Now()
	
	// Record MCP tool call metrics
	if fs.metrics != nil {
		fs.metrics.RecordMCPToolCall(ctx, "fetch")
		fs.metrics.RecordConcurrentRequestsChange(ctx, 1)
		defer fs.metrics.RecordConcurrentRequestsChange(ctx, -1)
	}

	// Convert to fetcher request
	fetchReq := &fetcher.FetchRequest{
		URL: params.URL,
		Raw: params.Raw,
	}

	if params.MaxLength != nil {
		fetchReq.MaxLength = params.MaxLength
	}

	if params.StartIndex != nil {
		fetchReq.StartIndex = params.StartIndex
	}

	// Fetch the content
	content, err := fs.fetcher.FetchURL(fetchReq)
	duration := time.Since(startTime)

	// Record metrics if enabled
	if fs.metrics != nil {
		result := "success"
		if err != nil {
			result = "error"
		}
		fs.metrics.RecordFetchOperation(ctx, params.URL, result, duration)
		fs.metrics.RecordMCPRequest(ctx, "tools/call", duration, err == nil)
		
		// Record response size metrics
		if err == nil {
			contentSize := int64(len(content))
			fs.metrics.RecordFetchResponseSize(ctx, params.URL, contentSize)
		}
	}

	if err != nil {
		return nil, nil, err
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: content}},
	}, nil, nil
}

// Start starts the MCP server following the MCP specification
func (fs *FetchServer) Start() error {
	fs.logServerStartup()

	switch fs.config.Transport {
	case config.TransportSSE:
		// For SSE, we need to create an HTTP server that handles SSE connections
		return fs.startSSEServer()

	case config.TransportStreamableHTTP:
		// For streamable HTTP, we need to create an HTTP server that handles streaming
		return fs.startStreamableHTTPServer()

	default:
		return fmt.Errorf("unsupported transport type: %s", fs.config.Transport)
	}
}

// startSSEServer starts the server with SSE transport
func (fs *FetchServer) startSSEServer() error {
	mux := http.NewServeMux()

	// Create SSE handler according to MCP specification
	sseHandler := mcp.NewSSEHandler(func(_ *http.Request) *mcp.Server {
		return fs.mcpServer
	}, &mcp.SSEOptions{})

	// Handle SSE endpoint
	if fs.traceHelper != nil {
		mux.Handle("/sse", otelhttp.NewHandler(sseHandler, "sse"))
		// HTTP POST endpoint for client-to-server communication
		mux.Handle("/messages", otelhttp.NewHandler(sseHandler, "messages"))
	} else {
		mux.Handle("/sse", sseHandler)
		// HTTP POST endpoint for client-to-server communication
		mux.Handle("/messages", sseHandler)
	}
	
	// Add Prometheus metrics endpoint if enabled
	if fs.config.EnablePrometheus && fs.telemetry != nil {
		mux.Handle("/metrics", fs.telemetry.PrometheusHandler())
		log.Printf("Prometheus metrics endpoint enabled at /metrics")
	}

	// Start HTTP server
	server := &http.Server{
		Addr:              ":" + strconv.Itoa(fs.config.Port),
		Handler:           mux,
		ReadHeaderTimeout: 30 * time.Second,
	}

	log.Printf("Server listening on %d", fs.config.Port)
	return server.ListenAndServe()
}

// startStreamableHTTPServer starts the server with streamable HTTP transport
func (fs *FetchServer) startStreamableHTTPServer() error {
	mux := http.NewServeMux()

	// Create streamable HTTP handler according to MCP specification
	streamableHandler := mcp.NewStreamableHTTPHandler(
		func(_ *http.Request) *mcp.Server {
			return fs.mcpServer
		},
		&mcp.StreamableHTTPOptions{
			// Configure any specific options here if needed
		},
	)

	// Handle the message endpoint
	if fs.traceHelper != nil {
		mux.Handle("/mcp", otelhttp.NewHandler(streamableHandler, "mcp"))
	} else {
		mux.Handle("/mcp", streamableHandler)
	}
	
	// Add Prometheus metrics endpoint if enabled
	if fs.config.EnablePrometheus && fs.telemetry != nil {
		mux.Handle("/metrics", fs.telemetry.PrometheusHandler())
		log.Printf("Prometheus metrics endpoint enabled at /metrics")
	}

	// Start HTTP server
	server := &http.Server{
		Addr:              ":" + strconv.Itoa(fs.config.Port),
		Handler:           mux,
		ReadHeaderTimeout: 30 * time.Second,
	}

	log.Printf("Server listening on %d", fs.config.Port)
	return server.ListenAndServe()
}

// logServerStartup prints startup information
func (fs *FetchServer) logServerStartup() {
	log.Printf("=== Starting MCP gofetch Server ===")
	log.Printf("Server port: %d", fs.config.Port)
	log.Printf("Transport: %s", fs.config.Transport)
	log.Printf("User agent: %s", fs.config.UserAgent)
	log.Printf("Ignore robots.txt: %v", fs.config.IgnoreRobots)
	if fs.config.ProxyURL != "" {
		log.Printf("Using proxy: %s", fs.config.ProxyURL)
	}
	log.Printf("Available tools: fetch")

	// Log endpoint based on transport
	switch fs.config.Transport {
	case config.TransportSSE:
		log.Printf("SSE endpoint (server-to-client): http://localhost:%d/sse", fs.config.Port)
		log.Printf("Messages endpoint (client-to-server): http://localhost:%d/messages", fs.config.Port)
	case config.TransportStreamableHTTP:
		log.Printf("MCP endpoint (streaming and commands): http://localhost:%d/mcp", fs.config.Port)
	}

	// Log observability features
	if fs.config.EnableOTelMetrics {
		log.Printf("OpenTelemetry metrics enabled")
	}
	if fs.config.EnableOTelTracing {
		log.Printf("OpenTelemetry tracing enabled")
	}
	if fs.config.EnablePrometheus {
		log.Printf("Prometheus metrics endpoint: http://localhost:%d/metrics", fs.config.Port)
	}

	log.Printf("=== Server starting ===")
}

// Shutdown gracefully shuts down the server and observability components
func (fs *FetchServer) Shutdown(ctx context.Context) error {
	if fs.telemetry != nil {
		log.Printf("Shutting down observability components...")
		return fs.telemetry.Shutdown(ctx)
	}
	return nil
}
