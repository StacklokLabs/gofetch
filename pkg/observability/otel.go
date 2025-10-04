// Package observability provides OpenTelemetry metrics and tracing setup
package observability

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	"github.com/stackloklabs/gofetch/pkg/config"
)

// ObservabilityConfig holds configuration for telemetry
type ObservabilityConfig struct {
	ServiceName    string
	ServiceVersion string
	Endpoint       string
	EnableMetrics  bool
	EnableTracing  bool
}

// Telemetry holds the telemetry providers and shutdown functions
type Telemetry struct {
	TracerProvider  trace.TracerProvider
	MeterProvider   metric.MeterProvider
	shutdownFuncs   []func(context.Context) error
}

// NewObservabilityConfig creates observability config from main config
func NewObservabilityConfig(cfg config.Config) ObservabilityConfig {
	return ObservabilityConfig{
		ServiceName:       cfg.OTelServiceName,
		ServiceVersion:    cfg.OTelServiceVersion,
		Endpoint:          cfg.OTelEndpoint,
		EnableMetrics:     cfg.EnableOTelMetrics,
		EnableTracing:     cfg.EnableOTelTracing,
	}
}

// Setup initializes OpenTelemetry telemetry providers
func Setup(ctx context.Context, obsConfig ObservabilityConfig) (*Telemetry, error) {
	tel := &Telemetry{}

	// Create resource with service information
	res, err := resource.New(ctx,
		resource.WithAttributes(
			attribute.String("service.name", obsConfig.ServiceName),
			attribute.String("service.version", obsConfig.ServiceVersion),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Setup tracing if enabled
	if obsConfig.EnableTracing {
		tracerProvider, shutdownFunc, err := setupTracing(ctx, res, obsConfig.Endpoint)
		if err != nil {
			return nil, fmt.Errorf("failed to setup tracing: %w", err)
		}
		tel.TracerProvider = tracerProvider
		tel.shutdownFuncs = append(tel.shutdownFuncs, shutdownFunc)
		
		// Set global tracer provider
		otel.SetTracerProvider(tracerProvider)
		
		// Set up W3C Trace Context propagation to extract/inject trace IDs from/to headers
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		))
		
		log.Printf("OpenTelemetry tracing initialized with endpoint: %s", obsConfig.Endpoint)
	}

	// Setup metrics if enabled
	if obsConfig.EnableMetrics {
		meterProvider, shutdownFunc, err := setupMetrics(ctx, res, obsConfig.Endpoint)
		if err != nil {
			return nil, fmt.Errorf("failed to setup metrics: %w", err)
		}
		tel.MeterProvider = meterProvider
		tel.shutdownFuncs = append(tel.shutdownFuncs, shutdownFunc)

		// Set global meter provider
		otel.SetMeterProvider(meterProvider)
		log.Printf("OpenTelemetry metrics initialized with endpoint: %s", obsConfig.Endpoint)
	}

	return tel, nil
}

// setupTracing configures OpenTelemetry tracing
func setupTracing(ctx context.Context, res *resource.Resource, endpoint string) (trace.TracerProvider, func(context.Context) error, error) {
	// Default to localhost:4318 if no endpoint is provided
	if endpoint == "" {
		endpoint = "localhost:4318"
		log.Printf("No OTLP endpoint specified for tracing, using default: %s", endpoint)
	}

	// Create OTLP HTTP exporter for traces
	traceExporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithInsecure(), // Use insecure for development
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Create trace provider
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	shutdownFunc := func(ctx context.Context) error {
		return tracerProvider.Shutdown(ctx)
	}

	return tracerProvider, shutdownFunc, nil
}

// setupMetrics configures OpenTelemetry metrics
func setupMetrics(ctx context.Context, res *resource.Resource, endpoint string) (metric.MeterProvider, func(context.Context) error, error) {
	// Default to localhost:4318 if no endpoint is provided (OTLP HTTP default port)
	if endpoint == "" {
		endpoint = "localhost:4318"
		log.Printf("No OTLP endpoint specified for metrics, using default: %s", endpoint)
	}

	// Create OTLP HTTP exporter for metrics
	metricExporter, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpoint(endpoint),
		otlpmetrichttp.WithInsecure(), // Use insecure for development
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create metric exporter: %w", err)
	}

	// Create meter provider
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(
				metricExporter,
				sdkmetric.WithInterval(30*time.Second),
			),
		),
		sdkmetric.WithResource(res),
	)

	shutdownFunc := func(ctx context.Context) error {
		return meterProvider.Shutdown(ctx)
	}

	return meterProvider, shutdownFunc, nil
}


// Shutdown gracefully shuts down all telemetry providers
func (t *Telemetry) Shutdown(ctx context.Context) error {
	for _, shutdown := range t.shutdownFuncs {
		if err := shutdown(ctx); err != nil {
			log.Printf("Error shutting down telemetry: %v", err)
		}
	}
	return nil
}