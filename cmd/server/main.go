// Package main is the entry point for the fetch MCP server.
package main

import (
	"context"
	"log"

	"github.com/stackloklabs/gofetch/pkg/config"
	"github.com/stackloklabs/gofetch/pkg/server"
)

func main() {
	// Parse configuration
	cfg := config.ParseFlags()

	// Create context for clean shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create and configure server
	fs := server.NewFetchServer(cfg)

	// Start server
	serverErrCh := make(chan error, 1)
	go func() {
		if err := fs.Start(); err != nil {
			log.Printf("Server error: %v", err)
			serverErrCh <- err
		}
	}()

	// Wait for error or shutdown signal
	select {
	case err := <-serverErrCh:
		log.Fatalf("Server failed to start: %v", err)
	case <-ctx.Done():
		log.Println("Shutdown signal received")
	}
}
