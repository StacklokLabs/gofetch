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

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stackloklabs/gofetch/pkg/config"
	"github.com/stackloklabs/gofetch/pkg/fetcher"
	"github.com/stackloklabs/gofetch/pkg/processor"
	"github.com/stackloklabs/gofetch/pkg/robots"
)

// FetchParams defines the input parameters for the fetch tool
type FetchParams struct {
	URL        string `json:"url" jsonschema:"URL to fetch"`
	MaxLength  *int   `json:"max_length,omitempty" jsonschema:"Maximum number of characters to return (default: no limit)"`
	StartIndex *int   `json:"start_index,omitempty" jsonschema:"Character index to start from when paging through truncated content"`
	Raw        bool   `json:"raw,omitempty" jsonschema:"Return the raw page content without converting HTML to markdown"`
}

// FetchServer represents the MCP server for fetching web content
type FetchServer struct {
	config    config.Config
	fetcher   *fetcher.HTTPFetcher
	mcpServer *mcp.Server
}

// NewFetchServer creates a new fetch server instance
func NewFetchServer(cfg config.Config) *FetchServer {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Configure proxy if provided
	if cfg.ProxyURL != "" {
		if proxyURLParsed, err := url.Parse(cfg.ProxyURL); err == nil {
			client.Transport = &http.Transport{
				Proxy: http.ProxyURL(proxyURLParsed),
			}
		}
	}

	// Create components
	robotsChecker := robots.NewChecker(cfg.UserAgent, cfg.IgnoreRobots, client)
	contentProcessor := processor.NewContentProcessor()
	httpFetcher := fetcher.NewHTTPFetcher(client, robotsChecker, contentProcessor, cfg.UserAgent)

	fs := &FetchServer{
		config:  cfg,
		fetcher: httpFetcher,
	}

	// Create MCP server with proper implementation details
	// Capabilities are automatically generated based on registered tools/resources
	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    config.ServerName,
		Version: config.ServerVersion,
	}, nil)

	fs.mcpServer = mcpServer

	// Setup tools
	fs.setupTools()

	return fs
}

// setupTools registers the fetch tool with the MCP server
func (fs *FetchServer) setupTools() {
	fetchTool := &mcp.Tool{
		Name:        "fetch",
		Title:       "Fetch URL",
		Description: "Fetches a URL from the internet and optionally extracts its contents as markdown.",
		InputSchema: fetchInputSchema(),
		// Fetching only reads remote content and never modifies the
		// environment, so mark the tool read-only. Without these hints,
		// clients fall back to the spec defaults (readOnlyHint=false,
		// destructiveHint=true) and treat the tool as a destructive write.
		Annotations: &mcp.ToolAnnotations{
			Title:          "Fetch URL",
			ReadOnlyHint:   true,
			IdempotentHint: true,
			OpenWorldHint:  boolPtr(true),
		},
	}

	mcp.AddTool(fs.mcpServer, fetchTool, fs.handleFetchTool)
}

// fetchInputSchema infers the input schema from FetchParams and adds bounds
// that struct tags can't express. The SDK validates arguments against this
// schema before the handler runs.
func fetchInputSchema() *jsonschema.Schema {
	schema, err := jsonschema.For[FetchParams](nil)
	if err != nil {
		// FetchParams is a static type, so inference can only fail on a
		// programming error.
		panic(fmt.Sprintf("inferring fetch input schema: %v", err))
	}
	schema.Properties["max_length"].Minimum = float64Ptr(1)
	schema.Properties["start_index"].Minimum = float64Ptr(0)
	return schema
}

func boolPtr(b bool) *bool { return &b }

func float64Ptr(f float64) *float64 { return &f }

// handleFetchTool processes fetch tool requests
func (fs *FetchServer) handleFetchTool(
	_ context.Context,
	_ *mcp.CallToolRequest,
	params FetchParams,
) (*mcp.CallToolResult, any, error) {
	log.Printf("Tool call received: fetch")

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
	mux.Handle("/sse", sseHandler)

	// HTTP POST endpoint for client-to-server communication
	mux.Handle("/messages", sseHandler)

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
			// The fetch tool keeps no state between calls, so run statelessly.
			// The SDK only serves MCP 2026-07-28 and later (which removed
			// protocol-level sessions) in stateless mode. Clients on earlier
			// protocol versions still work; they just don't get an
			// Mcp-Session-Id, which the earlier specs make optional.
			Stateless: true,
		},
	)

	// Handle the message endpoint
	mux.Handle("/mcp", streamableHandler)

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
		log.Printf("WARNING: the HTTP+SSE transport is deprecated by the MCP specification " +
			"and does not support protocol version 2026-07-28 or later; use streamable-http instead")
		log.Printf("SSE endpoint (server-to-client): http://localhost:%d/sse", fs.config.Port)
		log.Printf("Messages endpoint (client-to-server): http://localhost:%d/messages", fs.config.Port)
	case config.TransportStreamableHTTP:
		log.Printf("MCP endpoint (streaming and commands): http://localhost:%d/mcp", fs.config.Port)
	}

	log.Printf("=== Server starting ===")
}
