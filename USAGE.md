# GoFetch MCP Server Usage Guide

This document provides examples of how to interact with the GoFetch MCP server using curl commands for both `sse` and `streamable-http` transport types.

## SSE

```bash
TRANSPORT=sse MCP_PORT=8080 ./gofetch
```

### Available Endpoints:
- **SSE endpoint**: `GET /sse` - for receiving server-to-client messages
- **Messages endpoint**: `POST /messages` - for sending client-to-server commands

SSE transport uses query parameters for session management, not headers.

### 1. Initialize Session (GET request to establish session)
```bash
# Start a persistent connection to receive messages
curl -H "Accept: text/event-stream" \
     -H "Mcp-Protocol-Version: 2025-06-18" \
     "http://localhost:8080/sse"
```

This will establish a session and return a URL with a session ID like:
```
data: /sse?sessionid=TE2NFYZFIWX2E6RIUJX7TNO7H5
```

### 2. Send Initialize Message (POST to messages endpoint)
```bash
# Use the session ID from step 1 and send to the messages endpoint
curl -X POST \
     -H "Content-Type: application/json" \
     -H "Mcp-Protocol-Version: 2025-06-18" \
     "http://localhost:8080/messages?sessionid=<sessionId>" \
     -d '{
       "jsonrpc": "2.0",
       "id": 1,
       "method": "initialize",
       "params": {
         "protocolVersion": "2025-06-18",
         "capabilities": {},
         "clientInfo": {
           "name": "curl-client",
           "version": "1.0.0"
         }
       }
     }'
```

### 3. List Available Tools
```bash
curl -X POST \
     -H "Content-Type: application/json" \
     -H "Mcp-Protocol-Version: 2025-06-18" \
     "http://localhost:8080/messages?sessionid=<sessionId>" \
     -d '{
       "jsonrpc": "2.0",
       "id": 2,
       "method": "tools/list"
     }'
```

### 4. Call Fetch Tool
```bash
curl -X POST \
     -H "Content-Type: application/json" \
     -H "Mcp-Protocol-Version: 2025-06-18" \
     "http://localhost:8080/messages?sessionid=<sessionId>" \
     -d '{
       "jsonrpc": "2.0",
       "id": 3,
       "method": "tools/call",
       "params": {
         "name": "fetch",
         "arguments": {
           "url": "https://example.com",
           "max_length": 1000
         }
       }
     }'
```

## Streamable-HTTP

```bash
TRANSPORT=streamable-http MCP_PORT=8080 ./gofetch
```

### Available Endpoints:
- **MCP endpoint**: `GET/POST /mcp` - for streaming server-to-client communication and client-to-server commands

### 1. Initialize Session and Get Session ID

First, initialize the session and capture the session ID from the response headers:

```bash
SESSION_ID=$(curl -s -D /dev/stderr \
  -X POST "http://localhost:8080/mcp" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -H "Mcp-Protocol-Version: 2025-06-18" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "initialize",
    "params": {
      "protocolVersion": "2025-06-18",
      "capabilities": {},
      "clientInfo": {
        "name": "curl-client",
        "version": "1.0.0"
      }
    }
  }' 2>&1 >/dev/null | grep "Mcp-Session-Id:" | cut -d' ' -f2 | tr -d '\r')

echo "Session ID: $SESSION_ID"
```

### 2. List Available Tools

```bash
curl -X POST "http://localhost:8080/mcp" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -H "Mcp-Protocol-Version: 2025-06-18" \
  -H "Mcp-Session-Id: $SESSION_ID" \
  -d '{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "tools/list"
  }'
```

### 3. Call Fetch Tool

```bash
curl -X POST "http://localhost:8080/mcp" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -H "Mcp-Protocol-Version: 2025-06-18" \
  -H "Mcp-Session-Id: $SESSION_ID" \
  -d '{
    "jsonrpc": "2.0",
    "id": 3,
    "method": "tools/call",
    "params": {
      "name": "fetch",
      "arguments": {
        "url": "https://example.com",
        "max_length": 1000,
        "raw": false
      }
    }
  }'
```

### 4. Stream Responses (for Streamable-HTTP)

To receive server-to-client messages and responses:

```bash
curl -X GET "http://localhost:8080/mcp" \
  -H "Accept: text/event-stream" \
  -H "Mcp-Protocol-Version: 2025-06-18" \
  -H "Mcp-Session-Id: $SESSION_ID"
```

## Fetch Tool Parameters

The fetch tool accepts the following parameters:

- `url` (required): The URL to fetch
- `max_length` (optional): Maximum number of characters to return
- `start_index` (optional): Start index for truncated content  
- `raw` (optional): If true, returns raw HTML instead of markdown


## Transport Differences

| Feature | Streamable HTTP (Modern) | HTTP+SSE (Legacy) |
|---------|-------------------------|-------------------|
| Endpoint | `GET/POST /mcp` for streaming responses and commands | `GET /sse` for streaming responses |
| Command Endpoint | Same as above | `POST /messages` for client commands |
| Session Identification | Uses **HTTP headers** for session management (`Mcp-Session-Id` header) | Uses **query parameters** in URL for session management (`?sessionid=...`) |
| Session Termination | `DELETE /mcp` | Connection close |