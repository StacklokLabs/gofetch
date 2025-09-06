// Package main is the entry point for the fetch MCP server.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/stackloklabs/gofetch/pkg/config"
	"github.com/stackloklabs/gofetch/pkg/server"
)

func main() {
	// Parse configuration
	cfg := config.ParseFlags()

	// Create and configure server
	fs := server.NewFetchServer(cfg)

	// Setup graceful shutdown handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// Start server
	serverErrCh := make(chan error, 1)
	go func() {
		if err := fs.Start(); err != nil {
			log.Printf("Server error: %v", err)
			serverErrCh <- err
		}
	}()

	// Wait for shutdown signal or server error
	select {
	case err := <-serverErrCh:
		log.Fatalf("Server failed to start: %v", err)
	case sig := <-sigCh:
		log.Printf("Shutdown signal received: %s", sig)
		
		// Create shutdown context with timeout
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()
		
		// Gracefully shutdown observability components
		if err := fs.Shutdown(shutdownCtx); err != nil {
			log.Printf("Error shutting down observability: %v", err)
		}
		
		log.Println("Server shutdown completed")
	}
}
