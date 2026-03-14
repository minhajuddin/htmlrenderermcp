---
cwd: /Users/minhajuddin/r/htmlrenderermcp
prompt: "Build an MCP server that takes HTML as input and provides a link to render it in a browser. Phase 1: HTML files. Needs MCP server + HTTP server. Go, single process, S3-compatible storage, proxy through HTTP server."
branch: main
timestamp: 2026-03-14T00:00:00Z
version: 0
source: ~/.claude/plans/precious-cuddling-spring.md
---

# HTML Renderer MCP Server - Phase 1 Plan

## Context

Build an MCP server that accepts HTML content from Claude (or any MCP client) and provides a URL to view it in a browser. This enables Claude to generate rich HTML visualizations, reports, and pages that users can immediately open and interact with.

Phase 1 focuses on HTML rendering only. Future phases will add JSON and Markdown support.

## Architecture

**Single Go process** running two servers concurrently:
- **MCP server** (stdio transport) - receives HTML from Claude
- **HTTP server** (TCP) - serves rendered HTML to browsers

**Storage**: S3-compatible (Minio, R2, etc.) with configurable endpoint. HTTP server proxies content from S3.

**Flow**: Claude sends HTML via MCP tool -> server uploads to S3 -> returns URL -> user opens URL -> HTTP server fetches from S3 and serves.

## Dependencies

- `github.com/mark3labs/mcp-go` - Go MCP SDK (stdio server via `NewStdioServer` + `Listen`)
- `github.com/aws/aws-sdk-go-v2` (config, credentials, service/s3) - S3 client
- `github.com/google/uuid` - unique IDs for renders
- Go 1.22+ stdlib `net/http` with `{id}` wildcard routing (no external router needed)

## File Structure

```
htmlrenderermcp/
  go.mod
  main.go          # Entrypoint: wires config, storage, MCP, HTTP; concurrent execution
  config.go        # Config struct + LoadConfig() from env vars
  s3client.go      # Storage struct wrapping S3 client: NewStorage, Upload, Fetch
  mcpserver.go     # SetupMCPServer + render_html tool definition & handler
  httpserver.go    # SetupHTTPServer with GET /render/{id} and GET /health
```

## Implementation Steps

### Step 1: `go.mod` - Project init
```bash
go mod init github.com/minhajuddin/htmlrenderermcp
go get github.com/mark3labs/mcp-go github.com/aws/aws-sdk-go-v2/config github.com/aws/aws-sdk-go-v2/credentials github.com/aws/aws-sdk-go-v2/service/s3 github.com/google/uuid
```

### Step 2: `config.go` - Configuration
- `Config` struct with S3 and HTTP settings
- `LoadConfig()` reads from environment variables with sensible defaults
- Required: `S3_ENDPOINT`, `S3_ACCESS_KEY_ID`, `S3_SECRET_ACCESS_KEY`
- Defaults: `S3_BUCKET=html-renders`, `S3_REGION=us-east-1`, `S3_USE_PATH_STYLE=true`, `HTTP_ADDR=:8080`, `BASE_URL=http://localhost:8080`

### Step 3: `s3client.go` - S3 storage layer
- `Storage` struct wrapping `*s3.Client` and bucket name
- `NewStorage(cfg)` - creates S3 client with custom endpoint, static credentials, path-style addressing
- `Upload(ctx, id, html)` - puts `renders/{id}.html` with `Content-Type: text/html; charset=utf-8`
- `Fetch(ctx, id)` - gets `renders/{id}.html`, returns bytes
- `isNotFound(err)` helper using `smithy.APIError` to detect `NoSuchKey`

### Step 4: `mcpserver.go` - MCP server + tool
- `SetupMCPServer(storage, baseURL)` returns `*server.MCPServer`
- Registers `render_html` tool with:
  - Description: explains the tool renders HTML and returns a viewable URL
  - Input: `html` string (required)
- Handler: generates UUID, uploads to S3, returns `{baseURL}/render/{id}`
- Errors returned as `mcp.NewToolResultError()` (not Go errors) to report to LLM gracefully

### Step 5: `httpserver.go` - HTTP server
- `SetupHTTPServer(storage, addr)` returns `*http.Server`
- `GET /render/{id}` - validates UUID format (prevents path traversal), fetches from S3, serves as `text/html`
- `GET /health` - returns 200 OK
- 404 for missing IDs, 400 for invalid IDs

### Step 6: `main.go` - Wire everything together
- Load config, create storage, set up both servers
- `log.SetOutput(os.Stderr)` - **critical**: stdout is reserved for MCP protocol
- HTTP server runs in a goroutine
- MCP stdio server runs on main goroutine via `stdioServer.Listen(ctx, os.Stdin, os.Stdout)`
- Shared `signal.NotifyContext` for SIGINT/SIGTERM
- Graceful shutdown: when MCP stdin closes, HTTP server gets `Shutdown()` with 5s timeout

## Key Design Decisions

1. **`NewStdioServer` + `Listen` over `ServeStdio`**: Gives us context control needed for running both servers. `ServeStdio` installs its own signal handler which would conflict.
2. **All logging to stderr**: stdout is the MCP protocol channel. Any stray output corrupts it.
3. **UUID validation on HTTP endpoint**: Prevents path traversal attacks against S3 keys.
4. **Tool errors as `NewToolResultError`**: Reports failures to the LLM via MCP protocol instead of crashing the server.

## Verification

1. **Local Minio setup**:
   ```bash
   docker run -p 9000:9000 -p 9001:9001 \
     -e MINIO_ROOT_USER=minioadmin -e MINIO_ROOT_PASSWORD=minioadmin \
     minio/minio server /data --console-address ":9001"
   aws --endpoint-url http://localhost:9000 s3 mb s3://html-renders
   ```

2. **Run the server**:
   ```bash
   S3_ENDPOINT=http://localhost:9000 S3_ACCESS_KEY_ID=minioadmin S3_SECRET_ACCESS_KEY=minioadmin go run .
   ```

3. **Test via MCP protocol on stdin** (send JSON-RPC initialize, then tools/call with HTML)

4. **Open returned URL in browser** - should render the HTML page

5. **Claude Code integration**: Add to MCP server config and have Claude use the `render_html` tool to generate and serve a page
