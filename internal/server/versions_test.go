package server

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cruciblehq/protocol/pkg/registry"
)

func TestListVersions(t *testing.T) {
	mock := &mockRegistry{
		listVersionsFn: func(ctx context.Context, namespace string, resource string) (*registry.VersionList, error) {
			return &registry.VersionList{
				Versions: []registry.VersionSummary{
					{String: "1.0.0"},
				},
			}, nil
		},
	}

	handler := NewHandler(mock)
	req := httptest.NewRequest("GET", "/namespaces/test/resources/widget/versions", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "1.0.0") {
		t.Errorf("expected response to contain version")
	}
}

func TestCreateVersion(t *testing.T) {
	mock := &mockRegistry{
		createVersionFn: func(ctx context.Context, namespace string, resource string, info registry.VersionInfo) (*registry.Version, error) {
			return &registry.Version{
				Namespace: namespace,
				Resource:  resource,
				String:    info.String,
				CreatedAt: 1234567890,
			}, nil
		},
	}

	handler := NewHandler(mock)
	body := `{"string":"1.0.0"}`
	req := httptest.NewRequest("POST", "/namespaces/test/resources/widget/versions", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/vnd.crucible.version-info.v0+json")
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}

	location := w.Header().Get("Location")
	if !strings.Contains(location, "/versions/1.0.0") {
		t.Errorf("expected Location header with version path, got %s", location)
	}
}

func TestReadVersion(t *testing.T) {
	mock := &mockRegistry{
		readVersionFn: func(ctx context.Context, namespace string, resource string, version string) (*registry.Version, error) {
			return &registry.Version{
				Namespace: namespace,
				Resource:  resource,
				String:    version,
				CreatedAt: 1234567890,
			}, nil
		},
	}

	handler := NewHandler(mock)
	req := httptest.NewRequest("GET", "/namespaces/test/resources/widget/versions/1.0.0", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "1.0.0") {
		t.Errorf("expected response to contain version")
	}
}

func TestUpdateVersion(t *testing.T) {
	mock := &mockRegistry{
		updateVersionFn: func(ctx context.Context, namespace string, resource string, version string, info registry.VersionInfo) (*registry.Version, error) {
			return &registry.Version{
				Namespace: namespace,
				Resource:  resource,
				String:    version,
				UpdatedAt: 1234567890,
			}, nil
		},
	}

	handler := NewHandler(mock)
	body := `{"version":"1.0.0","description":"Updated description"}`
	req := httptest.NewRequest("PUT", "/namespaces/test/resources/widget/versions/1.0.0", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/vnd.crucible.version-info.v0+json")
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestDeleteVersion(t *testing.T) {
	mock := &mockRegistry{
		deleteVersionFn: func(ctx context.Context, namespace string, resource string, version string) error {
			return nil
		},
	}

	handler := NewHandler(mock)
	req := httptest.NewRequest("DELETE", "/namespaces/test/resources/widget/versions/1.0.0", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", w.Code)
	}
}

func TestUploadArchive(t *testing.T) {
	mock := &mockRegistry{
		uploadArchiveFn: func(ctx context.Context, namespace string, resource string, version string, archive io.Reader) (*registry.Version, error) {
			return &registry.Version{Namespace: namespace, Resource: resource, String: version}, nil
		},
	}

	handler := NewHandler(mock)
	body := bytes.NewReader([]byte("archive data"))
	req := httptest.NewRequest("PUT", "/namespaces/test/resources/widget/versions/1.0.0/archive", body)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestDownloadArchive(t *testing.T) {
	mock := &mockRegistry{
		downloadArchiveFn: func(ctx context.Context, namespace string, resource string, version string) (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader("archive data")), nil
		},
	}

	handler := NewHandler(mock)
	req := httptest.NewRequest("GET", "/namespaces/test/resources/widget/versions/1.0.0/archive", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != string(registry.MediaTypeArchive) {
		t.Errorf("expected Content-Type %s, got %s", registry.MediaTypeArchive, contentType)
	}

	contentDisposition := w.Header().Get("Content-Disposition")
	if !strings.Contains(contentDisposition, "widget-1.0.0.tar.zst") {
		t.Errorf("expected Content-Disposition with filename, got %s", contentDisposition)
	}

	if w.Body.String() != "archive data" {
		t.Errorf("expected archive data in response body")
	}
}

func TestDownloadArchiveNotFound(t *testing.T) {
	mock := &mockRegistry{
		downloadArchiveFn: func(ctx context.Context, namespace string, resource string, version string) (io.ReadCloser, error) {
			return nil, &registry.Error{
				Code:    registry.ErrorCodeNotFound,
				Message: "archive not found",
			}
		},
	}

	handler := NewHandler(mock)
	req := httptest.NewRequest("GET", "/namespaces/test/resources/widget/versions/1.0.0/archive", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}
