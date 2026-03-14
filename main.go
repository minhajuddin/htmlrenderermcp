package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mark3labs/mcp-go/server"
)

func main() {
	log.SetOutput(os.Stderr)

	cfg := LoadConfig()

	var store Storage
	var err error

	switch cfg.StorageBackend {
	case "disk":
		store, err = NewDiskStorage(cfg.DiskStoragePath)
	case "s3":
		store, err = NewS3Storage(cfg)
	default:
		log.Fatalf("unknown storage backend: %s", cfg.StorageBackend)
	}
	if err != nil {
		log.Fatalf("failed to initialize storage: %v", err)
	}

	mcpServer := SetupMCPServer(store, cfg.BaseURL, cfg.AutoOpen)
	httpServer := SetupHTTPServer(store, cfg.HTTPAddr, cfg.BaseURL)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("HTTP server listening on %s", cfg.HTTPAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	stdioServer := server.NewStdioServer(mcpServer)
	if err := stdioServer.Listen(ctx, os.Stdin, os.Stdout); err != nil {
		log.Printf("MCP server stopped: %v", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}
}
