package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cruciblehq/hub/internal/server"
	"github.com/cruciblehq/hub/internal/store"
	"github.com/cruciblehq/utils-go/crex"
	"github.com/cruciblehq/utils-go/file"
	"github.com/cruciblehq/utils-go/httpx"
	_ "modernc.org/sqlite"
)

const (
	defaultDBPath      = "./hub.db"        // Default SQLite database path.
	defaultArchiveRoot = "./archives"      // Default directory for archive storage.
	defaultPort        = "8080"            // Default HTTP server port.
	envDBPath          = "DB_PATH"         // Environment variable that overrides the SQLite database path.
	envArchiveRoot     = "ARCHIVE_ROOT"    // Environment variable that overrides the archive storage directory.
	envPort            = "PORT"            // Environment variable that overrides the HTTP server port.
	readHeaderTimeout  = 15 * time.Second  // Longest time a client may take to send the request headers.
	idleTimeout        = 120 * time.Second // Longest time an idle keep-alive connection stays open.
	shutdownTimeout    = 10 * time.Second  // Longest time to wait for open requests when shutting down.
)

func dbPath() string {
	if p := os.Getenv(envDBPath); p != "" {
		return p
	}
	return defaultDBPath
}

func archiveRoot() string {
	if p := os.Getenv(envArchiveRoot); p != "" {
		return p
	}
	return defaultArchiveRoot
}

func port() string {
	if p := os.Getenv(envPort); p != "" {
		return p
	}
	return defaultPort
}

func logger() *slog.Logger {
	return slog.New(crex.NewHandler().SetFormatter(&crex.JSONFormatter{Verbose: true}))
}

func main() {
	logger := logger()

	db := openDB(logger)
	defer db.Close()
	reg := openStore(logger, db)

	// Serve until an interrupt signal, then shut down gracefully.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	srv := newServer(server.NewHandler(reg))
	logger.Info("starting hub server", "addr", srv.Addr, "db", dbPath(), "archives", archiveRoot())
	if err := httpx.Serve(ctx, srv, shutdownTimeout); err != nil {
		logger.Error("server failed", "error", err)
		os.Exit(1)
	}

	logger.Info("server exited")
}

// Opens the SQLite database, exiting the process on failure.
func openDB(logger *slog.Logger) *sql.DB {
	db, err := sql.Open("sqlite", fmt.Sprintf("%s?_pragma=foreign_keys(1)", dbPath()))
	if err != nil {
		logger.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	return db
}

// Creates the archive directory and the registry store on db, exiting the
// process on failure.
func openStore(logger *slog.Logger, db *sql.DB) *store.SQLStore {
	root := archiveRoot()
	if err := os.MkdirAll(root, file.DefaultDirMode); err != nil {
		logger.Error("failed to create archive root", "path", root, "error", err)
		os.Exit(1)
	}
	reg, err := store.New(context.Background(), db, root, logger)
	if err != nil {
		logger.Error("failed to create registry", "error", err)
		os.Exit(1)
	}
	return reg
}

// Builds the HTTP server for handler on the configured port.
func newServer(handler http.Handler) *http.Server {
	return &http.Server{
		Addr:    fmt.Sprintf(":%s", port()),
		Handler: handler,

		// ReadTimeout and WriteTimeout are intentionally left unset (0,
		// unlimited) because archive uploads and downloads can be large and
		// slow. ReadHeaderTimeout still bounds slow-header attacks.
		ReadHeaderTimeout: readHeaderTimeout,
		IdleTimeout:       idleTimeout,
	}
}
