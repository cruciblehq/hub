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

func TestListResources(t *testing.T) {
	mock := &mockRegistry{
		listResourcesFn: func(ctx context.Context, namespace string) (*registry.ResourceList, error) {
			return &registry.ResourceList{
				Resources: []registry.ResourceSummary{
					{Name: "widget", Type: "widget", Description: "Test widget"},
				},
			}, nil
		},
	}

	handler := NewHandler(mock)
	req := httptest.NewRequest("GET", "/namespaces/test/resources", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "widget") {
		t.Errorf("expected response to contain resource name")
	}
}

func TestCreateResource(t *testing.T) {
	mock := &mockRegistry{
		createResourceFn: func(ctx context.Context, namespace string, info registry.ResourceInfo) (*registry.Resource, error) {
			return &registry.Resource{
				Namespace:   namespace,
				Name:        info.Name,
				Type:        info.Type,
				Description: info.Description,
				CreatedAt:   1234567890,
			}, nil
		},
	}

	handler := NewHandler(mock)
	body := `{"name":"widget","type":"widget","description":"Test widget"}`
	req := httptest.NewRequest("POST", "/namespaces/test/resources", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/vnd.crucible.resource-info.v0+json")
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}

	location := w.Header().Get("Location")
	if !strings.Contains(location, "/resources/widget") {
		t.Errorf("expected Location header with resource path, got %s", location)
	}
}

func TestReadResource(t *testing.T) {
	mock := &mockRegistry{
		readResourceFn: func(ctx context.Context, namespace string, resource string) (*registry.Resource, error) {
			return &registry.Resource{
				Namespace:   namespace,
				Name:        resource,
				Type:        "widget",
				Description: "Test widget",
				CreatedAt:   1234567890,
			}, nil
		},
	}

	handler := NewHandler(mock)
	req := httptest.NewRequest("GET", "/namespaces/test/resources/widget", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "widget") {
		t.Errorf("expected response to contain resource name")
	}
}

func TestUpdateResource(t *testing.T) {
	mock := &mockRegistry{
		updateResourceFn: func(ctx context.Context, namespace string, resource string, info registry.ResourceInfo) (*registry.Resource, error) {
			return &registry.Resource{
				Namespace:   namespace,
				Name:        resource,
				Type:        info.Type,
				Description: info.Description,
				CreatedAt:   1234567890,
			}, nil
		},
	}

	handler := NewHandler(mock)
	body := `{"name":"widget","type":"widget","description":"Updated description"}`
	req := httptest.NewRequest("PUT", "/namespaces/test/resources/widget", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/vnd.crucible.resource-info.v0+json")
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestDeleteResource(t *testing.T) {
	mock := &mockRegistry{
		deleteResourceFn: func(ctx context.Context, namespace string, resource string) error {
			return nil
		},
	}

	handler := NewHandler(mock)
	req := httptest.NewRequest("DELETE", "/namespaces/test/resources/widget", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", w.Code)
	}
}
