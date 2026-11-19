// Package main is the entry point for the fetch MCP server.
package main

import (
	"context"
	"log"
	"runtime/debug"
	"strings"

	"github.com/stackloklabs/gofetch/pkg/config"
	"github.com/stackloklabs/gofetch/pkg/server"
)

// version is set at build time with -X main.version=<version>. See .ko.yaml.
var version = config.DefaultVersion

// resolveVersion returns the server version to report to clients. It prefers
// the version set by ldflags. Otherwise it falls back to the module version
// recorded by `go install module@version`, and finally to "dev".
func resolveVersion(ldflagsVersion string, readBuildInfo func() (*debug.BuildInfo, bool)) string {
	if ldflagsVersion != "" && ldflagsVersion != config.DefaultVersion {
		return ldflagsVersion
	}
	if info, ok := readBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return strings.TrimPrefix(info.Main.Version, "v")
	}
	return config.DefaultVersion
}

func main() {
	// Parse configuration
	cfg := config.ParseFlags()
	cfg.Version = resolveVersion(version, debug.ReadBuildInfo)

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
