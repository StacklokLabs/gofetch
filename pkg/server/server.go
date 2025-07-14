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

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stackloklabs/gofetch/pkg/config"
	"github.com/stackloklabs/gofetch/pkg/fetcher"
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

	// Create MCP server with proper implementation details
	// Capabilities are automatically generated based on registered tools/resources
	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    config.ServerName,
		Version: config.ServerVersion,
	}, nil)

	fs := &FetchServer{
		config:    cfg,
		fetcher:   httpFetcher,
		mcpServer: mcpServer,
	}

	// Setup tools
	fs.setupTools()

	return fs
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
	_ context.Context,
	_ *mcp.ServerSession,
	params *mcp.CallToolParamsFor[FetchParams],
) (*mcp.CallToolResultFor[any], error) {
	log.Printf("Tool call received: fetch")

	// Convert to fetcher request
	fetchReq := &fetcher.FetchRequest{
		URL: params.Arguments.URL,
		Raw: params.Arguments.Raw,
	}

	if params.Arguments.MaxLength != nil {
		fetchReq.MaxLength = params.Arguments.MaxLength
	}

	if params.Arguments.StartIndex != nil {
		fetchReq.StartIndex = params.Arguments.StartIndex
	}

	// Fetch the content
	content, err := fs.fetcher.FetchURL(fetchReq)
	if err != nil {
		return nil, err
	}

	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{&mcp.TextContent{Text: content}},
	}, nil
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
	})

	// Handle SSE endpoint
	mux.Handle("/sse", sseHandler)

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
		log.Printf("SSE endpoint: http://localhost:%d/sse", fs.config.Port)
	case config.TransportStreamableHTTP:
		log.Printf("Message endpoint: http://localhost:%d/mcp", fs.config.Port)
	}

	log.Printf("=== Server starting ===")
}
