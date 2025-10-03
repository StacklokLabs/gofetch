#!/bin/bash

# Integration test script for gofetch MCP server
# Tests SSE and streamable-http transports using client binary
set -e

echo "Running integration tests for gofetch MCP server..."

# Test 1: Build the image using task build-image
echo "🏗️ Building Docker image using task build-image..."
task build-image
if [ $? -eq 0 ]; then
    echo "✓ Docker image built successfully using task build-image"
else
    echo "✗ Failed to build Docker image using task build-image"
    exit 1
fi

IMAGE_NAME="ghcr.io/stackloklabs/gofetch/server:latest"

cleanup() {
    docker rm -f gofetch-sse-endpoints-test > /dev/null 2>&1 || true
    docker rm -f gofetch-http-endpoints-test > /dev/null 2>&1 || true
}
trap cleanup EXIT

check_status() {
    local method=$1
    local url=$2
    local data=$3
    local header=$4

    if [ "$method" = "GET" ]; then
        curl -s -o /dev/null -m 5 -w "%{http_code}" ${header} "$url"
    else
        curl -s -o /dev/null -m 5 -w "%{http_code}" -X POST -H 'Content-Type: application/json' ${header} -d "${data}" "$url"
    fi
}

###############################################################
# SSE transport endpoint checks
###############################################################
echo ""
echo "🌊 ========== SSE ENDPOINT CHECKS =========="
docker run --rm -d --name gofetch-sse-endpoints-test -p 8080:8080 "$IMAGE_NAME" --transport sse --port 8080 > /dev/null 2>&1

if docker ps | grep -q gofetch-sse-endpoints-test; then
    echo "✓ SSE container started on port 8080"
else
    echo "✗ Failed to start SSE container"
    exit 1
fi

echo "🔎 Checking /sse (GET)"
# For SSE, the connection stays open; curl may timeout with exit 28. Capture headers and validate.
SSE_HEADERS=$(curl -sS -m 5 -D - -o /dev/null -H 'Accept: text/event-stream' http://localhost:8080/sse || true)
if echo "$SSE_HEADERS" | grep -qiE '^HTTP/[^ ]+ 200' && echo "$SSE_HEADERS" | grep -qi 'content-type: *text/event-stream'; then
    echo "✓ /sse endpoint reachable (200, text/event-stream)"
else
    echo "! /sse endpoint did not return expected headers"
    echo "$SSE_HEADERS" | sed 's/^/H: /'
    exit 1
fi

echo "🔎 Checking /messages (POST)"
MSG_STATUS=$(check_status POST "http://localhost:8080/messages" '{}' "" || true)
if [ "$MSG_STATUS" = "200" ] || [ "$MSG_STATUS" = "204" ] || [ "$MSG_STATUS" = "400" ]; then
    echo "✓ /messages endpoint reachable ($MSG_STATUS)"
else
    echo "! /messages endpoint not reachable (status: $MSG_STATUS)"
    exit 1
fi

docker rm -f gofetch-sse-endpoints-test > /dev/null 2>&1 || true
echo "✓ SSE container shut down"

###############################################################
# Streamable HTTP transport endpoint checks
###############################################################
echo ""
echo "🌐 ========== STREAMABLE-HTTP ENDPOINT CHECKS =========="
docker run --rm -d --name gofetch-http-endpoints-test -p 8081:8081 "$IMAGE_NAME" --transport streamable-http --port 8081 > /dev/null 2>&1

if docker ps | grep -q gofetch-http-endpoints-test; then
    echo "✓ Streamable HTTP container started on port 8081"
else
    echo "✗ Failed to start Streamable HTTP container"
    exit 1
fi

echo "🔎 Checking /mcp (GET)"
# GET without session should return 405 according to MCP spec for stateful servers
MCP_GET_STATUS=$(curl -s -o /dev/null -m 5 -w "%{http_code}" -H 'Accept: text/event-stream' "http://localhost:8081/mcp" 2>/dev/null || echo "000")
if [ "$MCP_GET_STATUS" = "200" ] || [ "$MCP_GET_STATUS" = "400" ] || [ "$MCP_GET_STATUS" = "405" ]; then
    echo "✓ /mcp endpoint reachable via GET ($MCP_GET_STATUS)"
else
    echo "! /mcp endpoint GET not reachable (status: $MCP_GET_STATUS)"
    exit 1
fi

echo "🔎 Checking /mcp (POST)"
MCP_POST_STATUS=$(check_status POST "http://localhost:8081/mcp" '{}' "" || true)
if [ "$MCP_POST_STATUS" = "200" ] || [ "$MCP_POST_STATUS" = "204" ] || [ "$MCP_POST_STATUS" = "400" ]; then
    echo "✓ /mcp endpoint reachable via POST ($MCP_POST_STATUS)"
else
    echo "! /mcp endpoint POST not reachable (status: $MCP_POST_STATUS)"
    exit 1
fi

docker rm -f gofetch-http-endpoints-test > /dev/null 2>&1 || true
echo "✓ Streamable HTTP container shut down"

echo "✅ Endpoint integration tests passed"


