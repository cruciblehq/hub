package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cruciblehq/hub/internal/server"
	"github.com/cruciblehq/protocol/pkg/registry"
	_ "modernc.org/sqlite"
)

const (
	defaultPort        = "8080"
	defaultDBPath      = "./hub.db"
	defaultArchiveRoot = "./archives"
)

func port() string {
	if p := os.Getenv("PORT"); p != "" {
		return p
	}
	return defaultPort
}

func dbPath() string {
	if p := os.Getenv("DB_PATH"); p != "" {
		return p
	}
	return defaultDBPath
}

func archiveRoot() string {
	if p := os.Getenv("ARCHIVE_ROOT"); p != "" {
		return p
	}
	return defaultArchiveRoot
}

func logger() *slog.Logger {
	return slog.Default()
}

func main() {

	// Setup logging
	logger := logger()

	// Open database
	dbPath := dbPath()
	db, err := sql.Open("sqlite", dbPath+"?_pragma=foreign_keys(1)")
	if err != nil {
		logger.Error("Failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Initialize registry
	ctx := context.Background()
	archiveRoot := archiveRoot()
	reg, err := registry.NewSQLRegistry(ctx, db, archiveRoot, logger)
	if err != nil {
		logger.Error("Failed to create registry", "error", err)
		os.Exit(1)
	}

	// Create HTTP handler
	handler := server.NewHandler(reg)

	// Get port from environment
	port := port()

	// Create HTTP server
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: handler,
	}

	// Start server in goroutine
	go func() {
		logger.Info("Starting hub server", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	logger.Info("Server exited")
}
