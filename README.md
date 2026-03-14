# HTML Renderer MCP Server

An MCP (Model Context Protocol) server that accepts HTML from Claude (or any MCP client) and serves it at a URL you can open in your browser. Lets Claude generate rich HTML visualizations, reports, dashboards, and interactive pages.

## How it works

1. Claude sends HTML via the `render_html` MCP tool
2. The server stores it (disk or S3) and returns a URL
3. Your browser opens the URL automatically
4. The HTTP server serves the rendered HTML

```
Claude ──MCP──> Server ──stores──> Disk/S3
                  │
Browser ──HTTP──> Server ──fetches──> Disk/S3
```

## Quick start

```bash
go build -o html-renderer .
./html-renderer
```

That's it. Uses disk storage at `/tmp/htmlrenderermcp` and serves on `:8080` by default. No external dependencies needed.

## Claude Code integration

Add to your MCP server config (`~/.claude/claude_desktop_config.json` or via `claude mcp add`):

```json
{
  "mcpServers": {
    "html-renderer": {
      "command": "/path/to/html-renderer"
    }
  }
}
```

Or with environment overrides:

```json
{
  "mcpServers": {
    "html-renderer": {
      "command": "/path/to/html-renderer",
      "env": {
        "HTTP_ADDR": ":9090",
        "BASE_URL": "http://localhost:9090",
        "AUTO_OPEN": "false"
      }
    }
  }
}
```

## MCP Tools

### `render_html`

Renders HTML content and returns a URL to view it.

| Parameter | Required | Description |
|-----------|----------|-------------|
| `html` | Yes | HTML content. Full documents are served as-is; fragments get wrapped in an HTML5 shell. |
| `title` | No | Page title (used if HTML lacks one) and label in the renders listing. |
| `id` | No | ID of an existing render to update in-place, instead of creating a new URL. |

Returns JSON: `{"url": "...", "id": "...", "title": "..."}`

### `list_renders`

Lists all rendered pages. No parameters. Returns a JSON array:

```json
[
  {
    "id": "054e9818-6230-4612-b8bc-328028ac6592",
    "title": "Sales Dashboard",
    "url": "http://localhost:8080/render/054e9818-6230-4612-b8bc-328028ac6592",
    "created_at": "2025-01-15T10:30:00Z"
  }
]
```

## HTTP Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /render/{id}` | Serves the rendered HTML page |
| `GET /renders` | HTML listing of all renders |
| `GET /health` | Returns `ok` |

## Configuration

All configuration is via environment variables.

### General

| Variable | Default | Description |
|----------|---------|-------------|
| `STORAGE_BACKEND` | `disk` | Storage backend: `disk` or `s3` |
| `HTTP_ADDR` | `:8080` | HTTP server listen address |
| `BASE_URL` | `http://localhost:8080` | Base URL for generated render links |
| `AUTO_OPEN` | `true` | Auto-open new renders in the default browser |

### Disk storage

| Variable | Default | Description |
|----------|---------|-------------|
| `DISK_STORAGE_PATH` | `/tmp/htmlrenderermcp` | Directory for stored renders |

### S3 storage

Set `STORAGE_BACKEND=s3` to use S3-compatible storage (Minio, R2, etc.).

| Variable | Default | Description |
|----------|---------|-------------|
| `S3_ENDPOINT` | *(required)* | S3 endpoint URL |
| `S3_ACCESS_KEY_ID` | *(required)* | Access key |
| `S3_SECRET_ACCESS_KEY` | *(required)* | Secret key |
| `S3_BUCKET` | `html-renders` | Bucket name |
| `S3_REGION` | `us-east-1` | Region |
| `S3_USE_PATH_STYLE` | `true` | Use path-style addressing (needed for Minio) |

#### Example with Minio

```bash
# Start Minio
docker run -p 9000:9000 -p 9001:9001 \
  -e MINIO_ROOT_USER=minioadmin -e MINIO_ROOT_PASSWORD=minioadmin \
  minio/minio server /data --console-address ":9001"

# Create the bucket
aws --endpoint-url http://localhost:9000 s3 mb s3://html-renders

# Run with S3 backend
STORAGE_BACKEND=s3 \
S3_ENDPOINT=http://localhost:9000 \
S3_ACCESS_KEY_ID=minioadmin \
S3_SECRET_ACCESS_KEY=minioadmin \
  ./html-renderer
```

## Features

- **Zero-config local usage** - works out of the box with disk storage
- **Auto-open browser** - new renders open in your default browser automatically
- **Update renders in-place** - pass an `id` to overwrite an existing render at the same URL
- **HTML fragment wrapping** - send `<div>Hello</div>` and it gets wrapped in a proper HTML5 document with charset, viewport, and title
- **Pluggable storage** - disk for local dev, S3-compatible for production
- **Renders listing** - browse all renders at `/renders` or query via `list_renders` tool
