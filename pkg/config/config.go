// Package config provides server configuration functionality.
package config

import (
	"flag"
	"os"
	"strconv"
)

// Constants
const (
	ServerName    = "fetch-server"
	ServerVersion = "1.0.0"
	DefaultUA     = "Mozilla/5.0 (compatible; MCPFetchBot/1.0)"
)

// Transport types
const (
	TransportSSE            = "sse"
	TransportStreamableHTTP = "streamable-http"
)

// Config holds the server configuration
type Config struct {
	Port         int
	UserAgent    string
	IgnoreRobots bool
	ProxyURL     string
	Transport    string
}

var transport string
var port int

// ParseFlags parses command line flags and returns configuration
func ParseFlags() Config {
	var config Config

	parseConfig(&config)

	config.Port = port
	config.Transport = transport

	// Set default user agent if not provided
	if config.UserAgent == "" {
		config.UserAgent = DefaultUA
	}

	return config
}

// parseConfig parses the command line flags and environment variables
// to set the transport and port for the MCP server
func parseConfig(config *Config) {
	flag.StringVar(&transport, "transport", "streamable-http", "Transport type: sse or streamable-http")
	flag.IntVar(&port, "port", 8080, "Port number for HTTP-based transports")
	flag.StringVar(&config.UserAgent, "user-agent", "", "Custom User-Agent string")
	flag.BoolVar(&config.IgnoreRobots, "ignore-robots-txt", false, "Ignore robots.txt rules")
	flag.StringVar(&config.ProxyURL, "proxy-url", "", "Proxy URL for requests")
	flag.Parse()

	if t, ok := os.LookupEnv("TRANSPORT"); ok {
		transport = t
	}
	if p, ok := os.LookupEnv("PORT"); ok {
		if intValue, err := strconv.Atoi(p); err == nil {
			port = intValue
		}
	}
}
