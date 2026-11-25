package store

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/cruciblehq/spec/registry"
)

func assertCode(t *testing.T, err error, want registry.ErrorCode) {
	t.Helper()
	var regErr *registry.Error
	if !errors.As(err, &regErr) || regErr.Code != want {
		t.Fatalf("error = %v, want code %s", err, want)
	}
}

func pushVersion(t *testing.T, s *SQLStore, ctx context.Context, version string, public bool) {
	t.Helper()
	if _, err := s.CreateVersion(ctx, "crucible", "machine-default", registry.VersionInfo{String: version}); err != nil {
		t.Fatalf("create %s: %v", version, err)
	}
	if _, err := s.UploadArchive(ctx, "crucible", "machine-default", version, bytes.NewReader([]byte(version))); err != nil {
		t.Fatalf("upload %s: %v", version, err)
	}
	if !public {
		return
	}
	if _, err := s.UpdateVersion(ctx, "crucible", "machine-default", version, registry.VersionInfo{String: version, Public: true}); err != nil {
		t.Fatalf("make %s public: %v", version, err)
	}
}

func TestUpdateVersionMakesPublic(t *testing.T) {
	s, _, ctx := newStoreForTest(t)
	pushVersion(t, s, ctx, "0.2.0", true)

	v, err := s.ReadVersion(ctx, "crucible", "machine-default", "0.2.0")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !v.Public {
		t.Fatal("Public = false, want true")
	}
}

func TestUpdateVersionPublicIsFinal(t *testing.T) {
	s, _, ctx := newStoreForTest(t)
	pushVersion(t, s, ctx, "0.2.0", true)

	_, err := s.UpdateVersion(ctx, "crucible", "machine-default", "0.2.0", registry.VersionInfo{String: "0.2.0"})
	assertCode(t, err, registry.ErrorCodeVersionPublic)
}

func TestUpdateVersionPrereleaseCannotBePublic(t *testing.T) {
	s, _, ctx := newStoreForTest(t)
	pushVersion(t, s, ctx, "0.2.0-dev.1", false)

	_, err := s.UpdateVersion(ctx, "crucible", "machine-default", "0.2.0-dev.1", registry.VersionInfo{String: "0.2.0-dev.1", Public: true})
	assertCode(t, err, registry.ErrorCodeBadRequest)
}

func TestUpdateVersionWithoutArchiveCannotBePublic(t *testing.T) {
	s, _, ctx := newStoreForTest(t)

	// newStoreForTest creates 0.1.0 without an archive.
	_, err := s.UpdateVersion(ctx, "crucible", "machine-default", "0.1.0", registry.VersionInfo{String: "0.1.0", Public: true})
	assertCode(t, err, registry.ErrorCodeBadRequest)
}

func TestUploadArchiveReplacesPrivateArchive(t *testing.T) {
	s, _, ctx := newStoreForTest(t)
	pushVersion(t, s, ctx, "0.2.0", false)

	v, err := s.UploadArchive(ctx, "crucible", "machine-default", "0.2.0", bytes.NewReader([]byte("changed")))
	if err != nil {
		t.Fatalf("second upload: %v", err)
	}
	if *v.Size != int64(len("changed")) {
		t.Fatalf("Size = %d, want %d", *v.Size, len("changed"))
	}
}

func TestUploadArchivePublicIsFinal(t *testing.T) {
	s, _, ctx := newStoreForTest(t)
	pushVersion(t, s, ctx, "0.2.0", true)

	_, err := s.UploadArchive(ctx, "crucible", "machine-default", "0.2.0", bytes.NewReader([]byte("changed")))
	assertCode(t, err, registry.ErrorCodeVersionPublic)
}

func TestCreateVersionBelowPublicOnSameLine(t *testing.T) {
	s, _, ctx := newStoreForTest(t)
	pushVersion(t, s, ctx, "0.2.3", true)

	_, err := s.CreateVersion(ctx, "crucible", "machine-default", registry.VersionInfo{String: "0.2.1"})
	assertCode(t, err, registry.ErrorCodeVersionSuperseded)
}

func TestCreateVersionPrereleaseBelowPublic(t *testing.T) {
	s, _, ctx := newStoreForTest(t)
	pushVersion(t, s, ctx, "0.2.0", true)

	_, err := s.CreateVersion(ctx, "crucible", "machine-default", registry.VersionInfo{String: "0.2.0-dev.1"})
	assertCode(t, err, registry.ErrorCodeVersionSuperseded)
}

func TestCreateVersionOnOlderLine(t *testing.T) {
	s, _, ctx := newStoreForTest(t)
	pushVersion(t, s, ctx, "1.0.0", true)

	if _, err := s.CreateVersion(ctx, "crucible", "machine-default", registry.VersionInfo{String: "0.9.4"}); err != nil {
		t.Fatalf("create 0.9.4: %v", err)
	}
}

func TestCreateVersionBelowPrivateVersion(t *testing.T) {
	s, _, ctx := newStoreForTest(t)
	pushVersion(t, s, ctx, "0.2.3", false)

	if _, err := s.CreateVersion(ctx, "crucible", "machine-default", registry.VersionInfo{String: "0.2.1"}); err != nil {
		t.Fatalf("create 0.2.1: %v", err)
	}
}

func TestUpdateVersionPublicBelowPublicOnSameLine(t *testing.T) {
	s, _, ctx := newStoreForTest(t)
	pushVersion(t, s, ctx, "0.2.1", false)
	pushVersion(t, s, ctx, "0.2.3", true)

	_, err := s.UpdateVersion(ctx, "crucible", "machine-default", "0.2.1", registry.VersionInfo{String: "0.2.1", Public: true})
	assertCode(t, err, registry.ErrorCodeVersionSuperseded)
}

func TestDeleteVersionPublicIsRefused(t *testing.T) {
	s, _, ctx := newStoreForTest(t)
	pushVersion(t, s, ctx, "0.2.0", true)

	err := s.DeleteVersion(ctx, "crucible", "machine-default", "0.2.0")
	assertCode(t, err, registry.ErrorCodeVersionPublic)
}

func TestDeleteVersionPrivate(t *testing.T) {
	s, _, ctx := newStoreForTest(t)
	pushVersion(t, s, ctx, "0.2.0", false)

	if err := s.DeleteVersion(ctx, "crucible", "machine-default", "0.2.0"); err != nil {
		t.Fatalf("delete: %v", err)
	}
}
