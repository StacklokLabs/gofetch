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

# Build client binary
echo "🔧 Building client binary..."
task build > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "✓ Client binary built successfully"
else
    echo "✗ Failed to build client binary"
    exit 1
fi

# Get the image name from ko build output
IMAGE_NAME="ghcr.io/stackloklabs/gofetch/server:latest"

cleanup() {
    docker rm -f gofetch-sse-test > /dev/null 2>&1 || true
    docker rm -f gofetch-http-test > /dev/null 2>&1 || true
}
trap cleanup EXIT

###################################################################
################## START - SSE TRANSPORT TESTING ##################
###################################################################
echo ""
echo "🌊 ========== SSE TRANSPORT TESTING ==========" 
echo "📡 Testing SSE transport on port 8080..."
docker run --rm -d --name gofetch-sse-test -p 8080:8080 $IMAGE_NAME --transport sse --port 8080 > /dev/null 2>&1
sleep 3

# Check if container is running
if docker ps | grep -q gofetch-sse-test; then
    echo "✓ SSE transport container started successfully on port 8080"
    
    # Test SSE endpoint with client binary
    echo "🌊 Testing SSE endpoint with client binary..."
    if yardstick-client -transport sse -address localhost -port 8080 -action info; then
        echo "✓ SSE client connection successful"
    else
        echo "! SSE client connection failed"
        exit 1
    fi
    
    # Test listing tools via SSE
    echo "📋 Testing tool listing via SSE..."
    if yardstick-client -transport sse -address localhost -port 8080 -action list-tools; then
        echo "✓ SSE tools listing successful"
    else
        echo "! SSE tools listing failed"
        exit 1
    fi
    
    echo "🔧 Testing tool calling via SSE..."
    if yardstick-client -transport sse -address localhost -port 8080 -action=call-tool -tool=fetch -args='{"url":"https://example.com"}' | grep -q "This domain is for use in illustrative examples in documents"; then
        echo "✅ SSE tool call returned expected output"
    else
        echo "! SSE tool call did not return expected output"
        exit 1
    fi
else
    echo "✗ SSE transport container failed to start on port 8080"
    exit 1
fi

# Cleanup SSE container
docker rm -f gofetch-sse-test > /dev/null 2>&1
echo "✓ SSE container shut down successfully"
###################################################################
################## END - SSE TRANSPORT TESTING ####################
###################################################################

###################################################################
############# START - StreamableHTTP TRANSPORT TESTING ############
###################################################################
echo ""
echo "🌐 ========== STREAMABLE-HTTP TRANSPORT TESTING ==========" 
echo "📡 Testing streamable-http transport on port 8081..."
docker run --rm -d --name gofetch-http-test -p 8081:8081 $IMAGE_NAME --transport streamable-http --port 8081 > /dev/null 2>&1
sleep 3

# Check if container is running
if docker ps | grep -q gofetch-http-test; then
    echo "✓ Streamable HTTP transport container started successfully on port 8081"
    
    # Test streamable-http endpoint with client binary
    echo "🌐 Testing streamable-http endpoint with client binary..."
    if yardstick-client -transport streamable-http -address localhost -port 8081 -action info; then
        echo "✓ Streamable HTTP client connection successful"
    else
        echo "! Streamable HTTP client connection failed"
        exit 1
    fi
    
    # Test listing tools via streamable-http
    echo "📋 Testing tool listing via streamable-http..."
    if yardstick-client -transport streamable-http -address localhost -port 8081 -action list-tools; then
        echo "✓ Streamable HTTP tools listing successful"
    else
        echo "! Streamable HTTP tools listing failed"
        exit 1
    fi
    
    echo "🔧 Testing tool calling via streamable-http..."
    if yardstick-client -transport streamable-http -address localhost -port 8081 -action=call-tool -tool=fetch -args='{"url":"https://example.com"}' | grep -q "This domain is for use in illustrative examples in documents"; then
        echo "✅ Streamable tool call returned expected output"
    else
        echo "! Streamable tool call did not return expected output"
        exit 1
    fi
else
    echo "✗ Streamable HTTP transport container failed to start on port 8081"
    exit 1
fi

# Cleanup streamable-http container
docker stop gofetch-http-test > /dev/null 2>&1
echo "✓ Streamable HTTP container shut down successfully"
###################################################################
############# END - StreamableHTTP TRANSPORT TESTING ##############
###################################################################
