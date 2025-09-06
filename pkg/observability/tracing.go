package observability

import (
	"context"
	"net/http"
	"net/url"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const (
	// Tracer name for the application
	TracerName = "gofetch"

	// Span names
	SpanHTTPRequest      = "http.request"
	SpanFetchURL         = "fetch.url"
	SpanProcessContent   = "content.process"
	SpanCheckRobots      = "robots.check"
	SpanHTMLToMarkdown   = "html.to_markdown"
)

// TraceHelper provides helper methods for distributed tracing
type TraceHelper struct {
	tracer trace.Tracer
}

// NewTraceHelper creates a new trace helper
func NewTraceHelper(serviceName string) *TraceHelper {
	tracer := otel.GetTracerProvider().Tracer(serviceName)
	return &TraceHelper{
		tracer: tracer,
	}
}

// StartHTTPSpan starts a new span for an HTTP request, extracting parent trace context from headers
func (th *TraceHelper) StartHTTPSpan(ctx context.Context, r *http.Request) (context.Context, trace.Span) {
	// Extract trace context from incoming HTTP headers (e.g., traceparent, tracestate)
	ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(r.Header))
	
	spanName := SpanHTTPRequest + " " + r.Method + " " + r.URL.Path

	ctx, span := th.tracer.Start(ctx, spanName,
		trace.WithAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.url", r.URL.String()),
			attribute.String("http.scheme", r.URL.Scheme),
			attribute.String("http.host", r.Host),
			attribute.String("http.user_agent", r.Header.Get("User-Agent")),
		),
		trace.WithSpanKind(trace.SpanKindServer),
	)

	return ctx, span
}

// FinishHTTPSpan finishes an HTTP span with response information
func (th *TraceHelper) FinishHTTPSpan(span trace.Span, statusCode int, err error) {
	span.SetAttributes(
		attribute.Int("http.status_code", statusCode),
	)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else if statusCode >= 400 {
		span.SetStatus(codes.Error, "HTTP error")
	} else {
		span.SetStatus(codes.Ok, "")
	}

	span.End()
}

// StartFetchSpan starts a new span for a URL fetch operation
func (th *TraceHelper) StartFetchSpan(ctx context.Context, fetchURL string) (context.Context, trace.Span) {
	ctx, span := th.tracer.Start(ctx, SpanFetchURL,
		trace.WithAttributes(
			attribute.String("fetch.url", fetchURL),
			attribute.String("fetch.host", extractHostFromURL(fetchURL)),
		),
		trace.WithSpanKind(trace.SpanKindClient),
	)

	return ctx, span
}

// FinishFetchSpan finishes a fetch span with result information
func (th *TraceHelper) FinishFetchSpan(span trace.Span, statusCode int, contentLength int, err error) {
	if statusCode > 0 {
		span.SetAttributes(attribute.Int("http.status_code", statusCode))
	}
	
	if contentLength > 0 {
		span.SetAttributes(attribute.Int("content.length", contentLength))
	}

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else if statusCode >= 400 {
		span.SetStatus(codes.Error, "Fetch failed")
	} else {
		span.SetStatus(codes.Ok, "Fetch successful")
	}

	span.End()
}

// StartProcessContentSpan starts a new span for content processing
func (th *TraceHelper) StartProcessContentSpan(ctx context.Context, contentType string) (context.Context, trace.Span) {
	ctx, span := th.tracer.Start(ctx, SpanProcessContent,
		trace.WithAttributes(
			attribute.String("content.type", contentType),
		),
	)

	return ctx, span
}

// FinishProcessContentSpan finishes a content processing span
func (th *TraceHelper) FinishProcessContentSpan(span trace.Span, inputLength, outputLength int, err error) {
	span.SetAttributes(
		attribute.Int("content.input_length", inputLength),
		attribute.Int("content.output_length", outputLength),
	)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "Content processed successfully")
	}

	span.End()
}

// StartRobotsCheckSpan starts a new span for robots.txt checking
func (th *TraceHelper) StartRobotsCheckSpan(ctx context.Context, checkURL string) (context.Context, trace.Span) {
	ctx, span := th.tracer.Start(ctx, SpanCheckRobots,
		trace.WithAttributes(
			attribute.String("robots.check_url", checkURL),
			attribute.String("robots.host", extractHostFromURL(checkURL)),
		),
	)

	return ctx, span
}

// FinishRobotsCheckSpan finishes a robots.txt check span
func (th *TraceHelper) FinishRobotsCheckSpan(span trace.Span, allowed bool, err error) {
	span.SetAttributes(
		attribute.Bool("robots.allowed", allowed),
	)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "Robots check completed")
	}

	span.End()
}

// AddSpanEvent adds a timestamped event to the current span
func (th *TraceHelper) AddSpanEvent(ctx context.Context, name string, attributes ...attribute.KeyValue) {
	span := trace.SpanFromContext(ctx)
	if span != nil && span.IsRecording() {
		span.AddEvent(name, trace.WithAttributes(attributes...))
	}
}

// extractHostFromURL extracts host from URL string
func extractHostFromURL(urlStr string) string {
	if parsedURL, err := url.Parse(urlStr); err == nil && parsedURL.Host != "" {
		return parsedURL.Host
	}
	return "unknown"
}

// InjectTraceContext injects trace context into outgoing HTTP request headers
func (th *TraceHelper) InjectTraceContext(ctx context.Context, r *http.Request) {
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(r.Header))
}

// GetTraceHelper returns a global trace helper instance
func GetTraceHelper(serviceName string) *TraceHelper {
	return NewTraceHelper(serviceName)
}