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
	
	// Observability configuration
	EnableOTelMetrics   bool
	EnableOTelTracing   bool
	OTelServiceName     string
	OTelServiceVersion  string
	OTelEndpoint        string
	EnablePrometheus    bool
	MetricsPort         int
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
	
	// Set default observability values
	if config.OTelServiceName == "" {
		config.OTelServiceName = ServerName
	}
	if config.OTelServiceVersion == "" {
		config.OTelServiceVersion = ServerVersion
	}
	if config.MetricsPort == 0 {
		config.MetricsPort = config.Port
	}

	return config
}

// parseConfig parses the command line flags and environment variables
// to set the transport and port for the MCP server
func parseConfig(config *Config) {
	// Server configuration flags
	flag.StringVar(&transport, "transport", "streamable-http", "Transport type: sse or streamable-http")
	flag.IntVar(&port, "port", 8080, "Port number for HTTP-based transports")
	flag.StringVar(&config.UserAgent, "user-agent", "", "Custom User-Agent string")
	flag.BoolVar(&config.IgnoreRobots, "ignore-robots-txt", false, "Ignore robots.txt rules")
	flag.StringVar(&config.ProxyURL, "proxy-url", "", "Proxy URL for requests")
	
	// Observability configuration flags
	flag.BoolVar(&config.EnableOTelMetrics, "enable-otel-metrics", false, "Enable OpenTelemetry metrics collection")
	flag.BoolVar(&config.EnableOTelTracing, "enable-otel-tracing", false, "Enable OpenTelemetry distributed tracing")
	flag.StringVar(&config.OTelServiceName, "otel-service-name", "", "Service name for OpenTelemetry (default: fetch-server)")
	flag.StringVar(&config.OTelServiceVersion, "otel-service-version", "", "Service version for OpenTelemetry (default: 1.0.0)")
	flag.StringVar(&config.OTelEndpoint, "otel-endpoint", "", "OTLP endpoint for exporting telemetry data")
	flag.BoolVar(&config.EnablePrometheus, "enable-prometheus", false, "Enable Prometheus metrics endpoint")
	flag.IntVar(&config.MetricsPort, "metrics-port", 0, "Port for metrics endpoint (default: same as main port)")
	
	flag.Parse()

	// Environment variable overrides
	if t, ok := os.LookupEnv("TRANSPORT"); ok {
		transport = t
	}
	if p, ok := os.LookupEnv("MCP_PORT"); ok {
		if intValue, err := strconv.Atoi(p); err == nil {
			port = intValue
		}
	}
	
	// Observability environment variables
	if enableMetrics, ok := os.LookupEnv("ENABLE_OTEL_METRICS"); ok {
		config.EnableOTelMetrics = enableMetrics == "true"
	}
	if enableTracing, ok := os.LookupEnv("ENABLE_OTEL_TRACING"); ok {
		config.EnableOTelTracing = enableTracing == "true"
	}
	if serviceName, ok := os.LookupEnv("OTEL_SERVICE_NAME"); ok {
		config.OTelServiceName = serviceName
	}
	if serviceVersion, ok := os.LookupEnv("OTEL_SERVICE_VERSION"); ok {
		config.OTelServiceVersion = serviceVersion
	}
	if endpoint, ok := os.LookupEnv("OTEL_EXPORTER_OTLP_ENDPOINT"); ok {
		config.OTelEndpoint = endpoint
	}
	if enablePrometheus, ok := os.LookupEnv("ENABLE_PROMETHEUS"); ok {
		config.EnablePrometheus = enablePrometheus == "true"
	}
	if metricsPort, ok := os.LookupEnv("METRICS_PORT"); ok {
		if intValue, err := strconv.Atoi(metricsPort); err == nil {
			config.MetricsPort = intValue
		}
	}
}
