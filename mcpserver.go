package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"os/exec"
	"runtime"
	"strings"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func SetupMCPServer(store Storage, baseURL string, autoOpen bool) *server.MCPServer {
	s := server.NewMCPServer(
		"html-renderer",
		"1.0.0",
	)

	renderTool := mcp.NewTool("render_html",
		mcp.WithDescription("Renders HTML content and returns a URL to view it in a browser. Use this to create rich visualizations, reports, dashboards, or any HTML page that the user can open and interact with."),
		mcp.WithString("html", mcp.Required(), mcp.Description("The HTML content to render. Can be a full HTML document or a fragment (fragments will be wrapped in a proper HTML5 document).")),
		mcp.WithString("title", mcp.Description("Optional title for the render. Used as the page title if the HTML doesn't already have one, and shown in the renders listing.")),
		mcp.WithString("id", mcp.Description("Optional ID of an existing render to update. If provided, overwrites the previous render at the same URL instead of creating a new one.")),
	)
	s.AddTool(renderTool, makeRenderHandler(store, baseURL, autoOpen))

	return s
}

func makeRenderHandler(store Storage, baseURL string, autoOpen bool) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := request.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("invalid arguments"), nil
		}

		htmlContent, ok := args["html"].(string)
		if !ok || htmlContent == "" {
			return mcp.NewToolResultError("html parameter is required"), nil
		}

		title, _ := args["title"].(string)
		id, _ := args["id"].(string)

		isUpdate := id != ""
		if !isUpdate {
			id = uuid.New().String()
		}

		htmlContent = wrapHTMLIfNeeded(htmlContent, title)

		meta := RenderMeta{
			ID:        id,
			Title:     title,
			CreatedAt: time.Now().UTC(),
		}

		if err := store.Upload(ctx, meta, []byte(htmlContent)); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to upload render: %v", err)), nil
		}

		url := fmt.Sprintf("%s/render/%s", baseURL, id)

		if autoOpen && !isUpdate {
			openBrowser(url)
		}

		response := map[string]string{
			"url": url,
			"id":  id,
		}
		if title != "" {
			response["title"] = title
		}

		jsonBytes, _ := json.Marshal(response)
		return mcp.NewToolResultText(string(jsonBytes)), nil
	}
}

func wrapHTMLIfNeeded(htmlContent string, title string) string {
	lower := strings.ToLower(htmlContent)
	if strings.Contains(lower, "<!doctype") || strings.Contains(lower, "<html") {
		return htmlContent
	}

	if title == "" {
		title = "Rendered Page"
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s</title>
</head>
<body>
%s
</body>
</html>`, html.EscapeString(title), htmlContent)
}

func openBrowser(url string) {
	var cmd string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "linux":
		cmd = "xdg-open"
	case "windows":
		cmd = "start"
	default:
		return
	}
	if err := exec.Command(cmd, url).Start(); err != nil {
		log.Printf("failed to open browser: %v", err)
	}
}
