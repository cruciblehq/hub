package server

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cruciblehq/protocol/pkg/registry"
)

func TestListNamespaces(t *testing.T) {
	mock := &mockRegistry{
		listNamespacesFn: func(ctx context.Context) (*registry.NamespaceList, error) {
			return &registry.NamespaceList{
				Namespaces: []registry.NamespaceSummary{
					{Name: "test", Description: "Test namespace"},
				},
			}, nil
		},
	}

	handler := NewHandler(mock)
	req := httptest.NewRequest("GET", "/namespaces", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "test") {
		t.Errorf("expected response to contain namespace name")
	}
}

func TestCreateNamespace(t *testing.T) {
	mock := &mockRegistry{
		createNamespaceFn: func(ctx context.Context, info registry.NamespaceInfo) (*registry.Namespace, error) {
			return &registry.Namespace{
				Name:        info.Name,
				Description: info.Description,
				CreatedAt:   1234567890,
			}, nil
		},
	}

	handler := NewHandler(mock)
	body := `{"name":"test","description":"Test namespace"}`
	req := httptest.NewRequest("POST", "/namespaces", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/vnd.crucible.namespace-info.v0+json")
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}

	location := w.Header().Get("Location")
	if !strings.Contains(location, "/namespaces/test") {
		t.Errorf("expected Location header with namespace path, got %s", location)
	}
}

func TestCreateNamespaceConflict(t *testing.T) {
	mock := &mockRegistry{
		createNamespaceFn: func(ctx context.Context, info registry.NamespaceInfo) (*registry.Namespace, error) {
			return nil, &registry.Error{
				Code:    registry.ErrorCodeNamespaceExists,
				Message: "namespace already exists",
			}
		},
	}

	handler := NewHandler(mock)
	body := `{"name":"test","description":"Test"}`
	req := httptest.NewRequest("POST", "/namespaces", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/vnd.crucible.namespace-info.v0+json")
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected status 409, got %d", w.Code)
	}
}

func TestReadNamespace(t *testing.T) {
	mock := &mockRegistry{
		readNamespaceFn: func(ctx context.Context, namespace string) (*registry.Namespace, error) {
			return &registry.Namespace{
				Name:        namespace,
				Description: "Test namespace",
				CreatedAt:   1234567890,
			}, nil
		},
	}

	handler := NewHandler(mock)
	req := httptest.NewRequest("GET", "/namespaces/test", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "test") {
		t.Errorf("expected response to contain namespace name")
	}
}

func TestReadNamespaceNotFound(t *testing.T) {
	mock := &mockRegistry{
		readNamespaceFn: func(ctx context.Context, namespace string) (*registry.Namespace, error) {
			return nil, &registry.Error{
				Code:    registry.ErrorCodeNotFound,
				Message: "namespace not found",
			}
		},
	}

	handler := NewHandler(mock)
	req := httptest.NewRequest("GET", "/namespaces/nonexistent", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestUpdateNamespace(t *testing.T) {
	mock := &mockRegistry{
		updateNamespaceFn: func(ctx context.Context, namespace string, info registry.NamespaceInfo) (*registry.Namespace, error) {
			return &registry.Namespace{
				Name:        namespace,
				Description: info.Description,
				CreatedAt:   1234567890,
			}, nil
		},
	}

	handler := NewHandler(mock)
	body := `{"name":"test","description":"Updated description"}`
	req := httptest.NewRequest("PUT", "/namespaces/test", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/vnd.crucible.namespace-info.v0+json")
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestDeleteNamespace(t *testing.T) {
	mock := &mockRegistry{
		deleteNamespaceFn: func(ctx context.Context, namespace string) error {
			return nil
		},
	}

	handler := NewHandler(mock)
	req := httptest.NewRequest("DELETE", "/namespaces/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", w.Code)
	}
}
