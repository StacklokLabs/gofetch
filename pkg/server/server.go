// Package server provides the MCP server implementation for gofetching web content.
package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/stackloklabs/gofetch/pkg/config"
	"github.com/stackloklabs/gofetch/pkg/fetcher"
	"github.com/stackloklabs/gofetch/pkg/processor"
	"github.com/stackloklabs/gofetch/pkg/robots"
)

// FetchServer represents the MCP server for fetching web content
type FetchServer struct {
	config    config.Config
	fetcher   *fetcher.HTTPFetcher
	mcpServer *server.MCPServer
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

	// Create MCP server
	mcpServer := server.NewMCPServer(config.ServerName, config.ServerVersion)

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
	fetchTool := mcp.NewTool("fetch",
		mcp.WithDescription("Fetches a URL from the internet and optionally extracts its contents as markdown."),
		mcp.WithString("url",
			mcp.Required(),
			mcp.Description("URL to fetch"),
			mcp.Pattern("^https?://.*"),
		),
		mcp.WithNumber("max_length",
			mcp.Description("Maximum number of characters to return."),
		),
		mcp.WithNumber("start_index",
			mcp.Description("Start index for truncated content."),
		),
		mcp.WithBoolean("raw",
			mcp.Description("Get the actual HTML content of the requested page, without simplification."),
		),
	)

	fs.mcpServer.AddTool(fetchTool, fs.handleFetchTool)
}

// handleFetchTool processes fetch tool requests
func (fs *FetchServer) handleFetchTool(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log.Printf("Tool call received: %s", request.Params.Name)

	// Parse request parameters
	fetchReq, err := fs.parseFetchRequest(request)
	if err != nil {
		log.Printf("Tool call failed - %v", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Fetch the content
	content, err := fs.fetcher.FetchURL(fetchReq)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(content), nil
}

// parseFetchRequest extracts and validates parameters from the MCP request
func (*FetchServer) parseFetchRequest(request mcp.CallToolRequest) (*fetcher.FetchRequest, error) {
	// Extract URL parameter (required)
	urlParam, err := request.RequireString("url")
	if err != nil {
		return nil, fmt.Errorf("URL is required")
	}

	// Extract optional parameters
	maxLength := request.GetInt("max_length", 0)
	startIndex := request.GetInt("start_index", 0)
	raw := request.GetBool("raw", false)

	fetchReq := &fetcher.FetchRequest{
		URL: urlParam,
		Raw: raw,
	}

	if maxLength > 0 {
		fetchReq.MaxLength = &maxLength
	}

	if startIndex > 0 {
		fetchReq.StartIndex = &startIndex
	}

	return fetchReq, nil
}

// Start starts the MCP server
func (fs *FetchServer) Start() error {
	fs.logServerStartup()

	sseServer := server.NewSSEServer(fs.mcpServer)
	return sseServer.Start(fs.config.Address)
}

// logServerStartup prints startup information
func (fs *FetchServer) logServerStartup() {
	log.Printf("=== Starting MCP gofetch Server ===")
	log.Printf("Server address: %s", fs.config.Address)
	log.Printf("User agent: %s", fs.config.UserAgent)
	log.Printf("Ignore robots.txt: %v", fs.config.IgnoreRobots)
	if fs.config.ProxyURL != "" {
		log.Printf("Using proxy: %s", fs.config.ProxyURL)
	}
	log.Printf("Available tools: fetch")
	log.Printf("SSE endpoint: http://localhost%s/sse", fs.config.Address)
	log.Printf("Message endpoint: http://localhost%s/message", fs.config.Address)
	log.Printf("=== Server starting ===")
}
