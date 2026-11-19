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

> **Deprecated:** The MCP specification deprecates the HTTP+SSE transport, and it doesn't support protocol version `2026-07-28` or later. Use `streamable-http` for new deployments.

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
- **MCP endpoint**: `POST /mcp` - for client-to-server requests; responses stream back on the same request

The server runs statelessly. It doesn't issue an `Mcp-Session-Id`, and every request stands on its own. `GET` and `DELETE` on `/mcp` return `405 Method Not Allowed`.

The server supports MCP protocol version `2026-07-28` as well as earlier versions back to `2024-11-05`.

### Protocol version 2026-07-28

Version `2026-07-28` has no `initialize` handshake. Each request carries its protocol version and client details in `params._meta`, and the `Mcp-Method` and `Mcp-Name` headers.

#### 1. Discover server capabilities (optional)

```bash
curl -X POST "http://localhost:8080/mcp" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -H "Mcp-Protocol-Version: 2026-07-28" \
  -H "Mcp-Method: server/discover" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "server/discover",
    "params": {
      "_meta": {
        "io.modelcontextprotocol/protocolVersion": "2026-07-28",
        "io.modelcontextprotocol/clientCapabilities": {},
        "io.modelcontextprotocol/clientInfo": {"name": "curl-client", "version": "1.0.0"}
      }
    }
  }'
```

#### 2. List available tools

```bash
curl -X POST "http://localhost:8080/mcp" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -H "Mcp-Protocol-Version: 2026-07-28" \
  -H "Mcp-Method: tools/list" \
  -d '{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "tools/list",
    "params": {
      "_meta": {
        "io.modelcontextprotocol/protocolVersion": "2026-07-28",
        "io.modelcontextprotocol/clientCapabilities": {},
        "io.modelcontextprotocol/clientInfo": {"name": "curl-client", "version": "1.0.0"}
      }
    }
  }'
```

#### 3. Call the fetch tool

```bash
curl -X POST "http://localhost:8080/mcp" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -H "Mcp-Protocol-Version: 2026-07-28" \
  -H "Mcp-Method: tools/call" \
  -H "Mcp-Name: fetch" \
  -d '{
    "jsonrpc": "2.0",
    "id": 3,
    "method": "tools/call",
    "params": {
      "_meta": {
        "io.modelcontextprotocol/protocolVersion": "2026-07-28",
        "io.modelcontextprotocol/clientCapabilities": {},
        "io.modelcontextprotocol/clientInfo": {"name": "curl-client", "version": "1.0.0"}
      },
      "name": "fetch",
      "arguments": {
        "url": "https://example.com",
        "max_length": 1000,
        "raw": false
      }
    }
  }'
```

### Earlier protocol versions

Clients on `2025-11-25` and earlier can still send `initialize` first. Because the server is stateless, no session ID comes back, so later requests don't need an `Mcp-Session-Id` header:

```bash
curl -X POST "http://localhost:8080/mcp" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json, text/event-stream" \
  -H "Mcp-Protocol-Version: 2025-11-25" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "fetch",
      "arguments": {"url": "https://example.com"}
    }
  }'
```

## Fetch Tool Parameters

The fetch tool accepts the following parameters:

- `url` (required): The URL to fetch
- `max_length` (optional, minimum 1): Maximum number of characters to return. If omitted, the full content is returned.
- `start_index` (optional, minimum 0): Character index to start from when paging through truncated content
- `raw` (optional): If true, returns the raw page content instead of markdown

The tool is annotated as read-only (`readOnlyHint: true`), idempotent, and open-world, so MCP clients don't treat it as a tool that modifies anything.


## Transport Differences

| Feature | Streamable HTTP (Modern) | HTTP+SSE (Legacy) |
|---------|-------------------------|-------------------|
| Endpoint | `POST /mcp` for requests; responses stream back on the same request | `GET /sse` for streaming responses |
| Command Endpoint | Same as above | `POST /messages` for client commands |
| Sessions | None (stateless) | Uses **query parameters** in URL for session management (`?sessionid=...`) |
| Protocol versions | `2024-11-05` through `2026-07-28` | `2024-11-05` through `2025-11-25` |