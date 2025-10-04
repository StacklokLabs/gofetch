# GoFetch Metrics Troubleshooting Guide

## Prerequisites

1. **Run the server with metrics enabled:**
   ```bash
   go run ./cmd/server --enable-otel-metrics
   ```
   Or specify an endpoint:
   ```bash
   go run ./cmd/server --enable-otel-metrics --otel-endpoint localhost:4318
   ```

2. **Have an OTLP collector running** to receive metrics (e.g., OpenTelemetry Collector, Jaeger, etc.)

3. **Configure Prometheus** to scrape metrics from your OTLP collector

## Debugging Steps

### 1. Check Server Logs

When starting the server, you should see:
```
Successfully initialized metrics with service name: fetch-server
OpenTelemetry metrics initialized with endpoint: localhost:4318
```

When fetching URLs, you should see:
```
Recorded domain fetch metric for URL: https://example.com
```

If you see:
```
Metrics not initialized - skipping domain fetch metric for URL: https://example.com
```
Then metrics are not being initialized properly.

### 2. Verify Metric Names in Prometheus

The metrics might be exported with different names. Check what metrics are actually available:

```bash
# Query Prometheus for all metric names
curl -s http://localhost:9090/api/v1/label/__name__/values | jq . | grep -i fetch

# Or check for specific patterns
curl -s http://localhost:9090/api/v1/label/__name__/values | jq . | grep -E "(domain|robots|network)"
```

### 3. Common Metric Name Transformations

OpenTelemetry might transform metric names. Look for these patterns:

| Expected Name | Possible Exported Names |
|--------------|-------------------------|
| `fetch_requests_by_domain` | `fetch_server_fetch_requests_by_domain` |
| | `fetch.requests.by.domain` |
| | `fetch_requests_by_domain_total` |
| `unique_domains_fetched` | `fetch_server_unique_domains_fetched` |
| | `unique.domains.fetched` |
| `robots_txt_violations_total` | `fetch_server_robots_txt_violations_total` |
| | `robots.txt.violations.total` |
| `network_errors_by_type` | `fetch_server_network_errors_by_type` |
| | `network.errors.by.type` |

### 4. Update Grafana Queries

Once you identify the actual metric names, update the Grafana dashboard queries. For example:

If metrics appear as `fetch_server_fetch_requests_by_domain`, update the query from:
```promql
sum by(domain) (rate(fetch_requests_by_domain[$__rate_interval]))
```
To:
```promql
sum by(domain) (rate(fetch_server_fetch_requests_by_domain[$__rate_interval]))
```

### 5. Test with Direct Prometheus Queries

Test if the metrics are being recorded:

```bash
# Check if any domain metrics exist
curl -G http://localhost:9090/api/v1/query \
  --data-urlencode 'query=fetch_requests_by_domain'

# Check with wildcards
curl -G http://localhost:9090/api/v1/query \
  --data-urlencode 'query={__name__=~".*domain.*"}'
```

### 6. OpenTelemetry Collector Configuration

If using OpenTelemetry Collector, ensure your config includes:

```yaml
receivers:
  otlp:
    protocols:
      http:
        endpoint: 0.0.0.0:4318

exporters:
  prometheus:
    endpoint: "0.0.0.0:8889"

service:
  pipelines:
    metrics:
      receivers: [otlp]
      exporters: [prometheus]
```

### 7. Generate Test Data

Make sure to actually fetch some URLs to generate metric data:

```bash
# Initialize session
SESSION_ID=$(curl -s -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -H "Mcp-Session-Id: test-session" \
  -d '{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {}}' \
  | jq -r '.result.sessionId')

# Fetch from different domains
for domain in example.com github.com google.com; do
  curl -X POST http://localhost:8080/mcp \
    -H "Content-Type: application/json" \
    -H "Mcp-Session-Id: $SESSION_ID" \
    -d "{\"jsonrpc\": \"2.0\", \"id\": 2, \"method\": \"tools/call\", \"params\": {\"name\": \"fetch\", \"arguments\": {\"url\": \"https://$domain\"}}}"
done
```

## Common Issues

1. **No OTLP endpoint configured**: The server now defaults to `localhost:4318` if not specified
2. **OTLP collector not running**: Metrics won't be exported if there's no collector to receive them
3. **Prometheus not configured to scrape**: Ensure Prometheus is configured to scrape from the OTLP collector's Prometheus exporter
4. **Metric name mismatches**: OpenTelemetry may add prefixes or change naming conventions

## Logs to Look For

With the recent updates, you should see these log messages:

- `"Successfully initialized metrics with service name: fetch-server"` - Metrics initialized
- `"Recorded domain fetch metric for URL: <url>"` - Domain metrics being recorded
- `"No OTLP endpoint specified for metrics, using default: localhost:4318"` - Default endpoint being used
- `"OpenTelemetry metrics initialized with endpoint: <endpoint>"` - Metrics setup complete