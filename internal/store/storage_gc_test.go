package store

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/cruciblehq/spec/registry"
)

func newStoreForTest(t *testing.T) (*SQLStore, *sql.DB, context.Context) {
	t.Helper()

	ctx := context.Background()
	root := t.TempDir()
	dbPath := filepath.Join(root, "hub.db")
	archiveRoot := filepath.Join(root, "archives")

	db, err := sql.Open("sqlite", fmt.Sprintf("%s?_pragma=foreign_keys(1)", dbPath))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	s, err := New(ctx, db, archiveRoot, nil)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	if _, err := s.CreateNamespace(ctx, registry.NamespaceInfo{Name: "crucible"}); err != nil {
		t.Fatalf("create namespace: %v", err)
	}
	if _, err := s.CreateResource(ctx, "crucible", registry.ResourceInfo{Name: "machine-default", Type: "machine"}); err != nil {
		t.Fatalf("create resource: %v", err)
	}
	if _, err := s.CreateVersion(ctx, "crucible", "machine-default", registry.VersionInfo{String: "0.1.0"}); err != nil {
		t.Fatalf("create version: %v", err)
	}

	return s, db, ctx
}

func TestUploadArchiveRemovesSupersededArchive(t *testing.T) {
	s, _, ctx := newStoreForTest(t)

	first, err := s.UploadArchive(ctx, "crucible", "machine-default", "0.1.0", bytes.NewReader(bytes.Repeat([]byte("a"), 1024)))
	if err != nil {
		t.Fatalf("first upload: %v", err)
	}
	if first.Archive == nil {
		t.Fatal("first upload archive path is nil")
	}
	firstPath := *first.Archive

	second, err := s.UploadArchive(ctx, "crucible", "machine-default", "0.1.0", bytes.NewReader(bytes.Repeat([]byte("b"), 2048)))
	if err != nil {
		t.Fatalf("second upload: %v", err)
	}
	if second.Archive == nil {
		t.Fatal("second upload archive path is nil")
	}
	if *second.Archive == firstPath {
		t.Fatal("expected replacement upload to change archive path")
	}

	if _, err := os.Stat(firstPath); !os.IsNotExist(err) {
		t.Fatalf("expected superseded archive to be removed, stat err=%v", err)
	}
}

func TestDeleteVersionRemovesArchiveFile(t *testing.T) {
	s, _, ctx := newStoreForTest(t)

	v, err := s.UploadArchive(ctx, "crucible", "machine-default", "0.1.0", bytes.NewReader(bytes.Repeat([]byte("z"), 4096)))
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	if v.Archive == nil {
		t.Fatal("upload archive path is nil")
	}
	path := *v.Archive

	if err := s.DeleteVersion(ctx, "crucible", "machine-default", "0.1.0"); err != nil {
		t.Fatalf("delete version: %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected archive to be removed after delete, stat err=%v", err)
	}
}

func TestStartupCleanupRemovesOrphanArchiveFiles(t *testing.T) {
	s, db, ctx := newStoreForTest(t)

	v, err := s.UploadArchive(ctx, "crucible", "machine-default", "0.1.0", bytes.NewReader(bytes.Repeat([]byte("x"), 1024)))
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	if v.Archive == nil {
		t.Fatal("upload archive path is nil")
	}
	path := *v.Archive

	if err := s.deleteVersion(ctx, "crucible", "machine-default", "0.1.0"); err != nil {
		t.Fatalf("raw delete version: %v", err)
	}

	fresh, err := New(ctx, db, s.archiveRoot, nil)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	_ = fresh

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected orphan archive to be removed on startup, stat err=%v", err)
	}
}
