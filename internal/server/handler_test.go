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

// Mock registry implementation for testing.
//
// Provides injectable function fields for each Registry method to enable
// customized behavior in tests. Default implementations return empty results
// or no-op operations when function fields are nil.
type mockRegistry struct {
	listNamespacesFn  func(ctx context.Context) (*registry.NamespaceList, error)
	createNamespaceFn func(ctx context.Context, info registry.NamespaceInfo) (*registry.Namespace, error)
	readNamespaceFn   func(ctx context.Context, namespace string) (*registry.Namespace, error)
	updateNamespaceFn func(ctx context.Context, namespace string, info registry.NamespaceInfo) (*registry.Namespace, error)
	deleteNamespaceFn func(ctx context.Context, namespace string) error
	listResourcesFn   func(ctx context.Context, namespace string) (*registry.ResourceList, error)
	createResourceFn  func(ctx context.Context, namespace string, info registry.ResourceInfo) (*registry.Resource, error)
	readResourceFn    func(ctx context.Context, namespace string, resource string) (*registry.Resource, error)
	updateResourceFn  func(ctx context.Context, namespace string, resource string, info registry.ResourceInfo) (*registry.Resource, error)
	deleteResourceFn  func(ctx context.Context, namespace string, resource string) error
	listVersionsFn    func(ctx context.Context, namespace string, resource string) (*registry.VersionList, error)
	createVersionFn   func(ctx context.Context, namespace string, resource string, info registry.VersionInfo) (*registry.Version, error)
	readVersionFn     func(ctx context.Context, namespace string, resource string, version string) (*registry.Version, error)
	updateVersionFn   func(ctx context.Context, namespace string, resource string, version string, info registry.VersionInfo) (*registry.Version, error)
	deleteVersionFn   func(ctx context.Context, namespace string, resource string, version string) error
	uploadArchiveFn   func(ctx context.Context, namespace string, resource string, version string, archive io.Reader) (*registry.Version, error)
	downloadArchiveFn func(ctx context.Context, namespace string, resource string, version string) (io.ReadCloser, error)
	listChannelsFn    func(ctx context.Context, namespace string, resource string) (*registry.ChannelList, error)
	createChannelFn   func(ctx context.Context, namespace string, resource string, info registry.ChannelInfo) (*registry.Channel, error)
	readChannelFn     func(ctx context.Context, namespace string, resource string, channel string) (*registry.Channel, error)
	updateChannelFn   func(ctx context.Context, namespace string, resource string, channel string, info registry.ChannelInfo) (*registry.Channel, error)
	deleteChannelFn   func(ctx context.Context, namespace string, resource string, channel string) error
}

func (m *mockRegistry) ListNamespaces(ctx context.Context) (*registry.NamespaceList, error) {
	if m.listNamespacesFn != nil {
		return m.listNamespacesFn(ctx)
	}
	return &registry.NamespaceList{Namespaces: []registry.NamespaceSummary{}}, nil
}

func (m *mockRegistry) CreateNamespace(ctx context.Context, info registry.NamespaceInfo) (*registry.Namespace, error) {
	if m.createNamespaceFn != nil {
		return m.createNamespaceFn(ctx, info)
	}
	return &registry.Namespace{Name: info.Name, Description: info.Description}, nil
}

func (m *mockRegistry) ReadNamespace(ctx context.Context, namespace string) (*registry.Namespace, error) {
	if m.readNamespaceFn != nil {
		return m.readNamespaceFn(ctx, namespace)
	}
	return &registry.Namespace{Name: namespace}, nil
}

func (m *mockRegistry) UpdateNamespace(ctx context.Context, namespace string, info registry.NamespaceInfo) (*registry.Namespace, error) {
	if m.updateNamespaceFn != nil {
		return m.updateNamespaceFn(ctx, namespace, info)
	}
	return &registry.Namespace{Name: namespace, Description: info.Description}, nil
}

func (m *mockRegistry) DeleteNamespace(ctx context.Context, namespace string) error {
	if m.deleteNamespaceFn != nil {
		return m.deleteNamespaceFn(ctx, namespace)
	}
	return nil
}

func (m *mockRegistry) ListResources(ctx context.Context, namespace string) (*registry.ResourceList, error) {
	if m.listResourcesFn != nil {
		return m.listResourcesFn(ctx, namespace)
	}
	return &registry.ResourceList{Resources: []registry.ResourceSummary{}}, nil
}

func (m *mockRegistry) CreateResource(ctx context.Context, namespace string, info registry.ResourceInfo) (*registry.Resource, error) {
	if m.createResourceFn != nil {
		return m.createResourceFn(ctx, namespace, info)
	}
	return &registry.Resource{Namespace: namespace, Name: info.Name, Description: info.Description}, nil
}

func (m *mockRegistry) ReadResource(ctx context.Context, namespace string, resource string) (*registry.Resource, error) {
	if m.readResourceFn != nil {
		return m.readResourceFn(ctx, namespace, resource)
	}
	return &registry.Resource{Namespace: namespace, Name: resource}, nil
}

func (m *mockRegistry) UpdateResource(ctx context.Context, namespace string, resource string, info registry.ResourceInfo) (*registry.Resource, error) {
	if m.updateResourceFn != nil {
		return m.updateResourceFn(ctx, namespace, resource, info)
	}
	return &registry.Resource{Namespace: namespace, Name: resource, Description: info.Description}, nil
}

func (m *mockRegistry) DeleteResource(ctx context.Context, namespace string, resource string) error {
	if m.deleteResourceFn != nil {
		return m.deleteResourceFn(ctx, namespace, resource)
	}
	return nil
}

func (m *mockRegistry) ListVersions(ctx context.Context, namespace string, resource string) (*registry.VersionList, error) {
	if m.listVersionsFn != nil {
		return m.listVersionsFn(ctx, namespace, resource)
	}
	return &registry.VersionList{Versions: []registry.VersionSummary{}}, nil
}

func (m *mockRegistry) CreateVersion(ctx context.Context, namespace string, resource string, info registry.VersionInfo) (*registry.Version, error) {
	if m.createVersionFn != nil {
		return m.createVersionFn(ctx, namespace, resource, info)
	}
	return &registry.Version{Namespace: namespace, Resource: resource, String: info.String}, nil
}

func (m *mockRegistry) ReadVersion(ctx context.Context, namespace string, resource string, version string) (*registry.Version, error) {
	if m.readVersionFn != nil {
		return m.readVersionFn(ctx, namespace, resource, version)
	}
	return &registry.Version{Namespace: namespace, Resource: resource, String: version}, nil
}

func (m *mockRegistry) UpdateVersion(ctx context.Context, namespace string, resource string, version string, info registry.VersionInfo) (*registry.Version, error) {
	if m.updateVersionFn != nil {
		return m.updateVersionFn(ctx, namespace, resource, version, info)
	}
	return &registry.Version{Namespace: namespace, Resource: resource, String: version}, nil
}

func (m *mockRegistry) DeleteVersion(ctx context.Context, namespace string, resource string, version string) error {
	if m.deleteVersionFn != nil {
		return m.deleteVersionFn(ctx, namespace, resource, version)
	}
	return nil
}

func (m *mockRegistry) UploadArchive(ctx context.Context, namespace string, resource string, version string, archive io.Reader) (*registry.Version, error) {
	if m.uploadArchiveFn != nil {
		return m.uploadArchiveFn(ctx, namespace, resource, version, archive)
	}
	return &registry.Version{Namespace: namespace, Resource: resource, String: version}, nil
}

func (m *mockRegistry) DownloadArchive(ctx context.Context, namespace string, resource string, version string) (io.ReadCloser, error) {
	if m.downloadArchiveFn != nil {
		return m.downloadArchiveFn(ctx, namespace, resource, version)
	}
	return io.NopCloser(strings.NewReader("mock archive data")), nil
}

func (m *mockRegistry) ListChannels(ctx context.Context, namespace string, resource string) (*registry.ChannelList, error) {
	if m.listChannelsFn != nil {
		return m.listChannelsFn(ctx, namespace, resource)
	}
	return &registry.ChannelList{Channels: []registry.ChannelSummary{}}, nil
}

func (m *mockRegistry) CreateChannel(ctx context.Context, namespace string, resource string, info registry.ChannelInfo) (*registry.Channel, error) {
	if m.createChannelFn != nil {
		return m.createChannelFn(ctx, namespace, resource, info)
	}
	return &registry.Channel{Namespace: namespace, Resource: resource, Name: info.Name, Version: registry.Version{String: info.Version}}, nil
}

func (m *mockRegistry) ReadChannel(ctx context.Context, namespace string, resource string, channel string) (*registry.Channel, error) {
	if m.readChannelFn != nil {
		return m.readChannelFn(ctx, namespace, resource, channel)
	}
	return &registry.Channel{Namespace: namespace, Resource: resource, Name: channel}, nil
}

func (m *mockRegistry) UpdateChannel(ctx context.Context, namespace string, resource string, channel string, info registry.ChannelInfo) (*registry.Channel, error) {
	if m.updateChannelFn != nil {
		return m.updateChannelFn(ctx, namespace, resource, channel, info)
	}
	return &registry.Channel{Namespace: namespace, Resource: resource, Name: channel, Version: registry.Version{String: info.Version}, Description: info.Description}, nil
}

func (m *mockRegistry) DeleteChannel(ctx context.Context, namespace string, resource string, channel string) error {
	if m.deleteChannelFn != nil {
		return m.deleteChannelFn(ctx, namespace, resource, channel)
	}
	return nil
}

func TestContentNegotiationJSON(t *testing.T) {
	mock := &mockRegistry{}
	handler := NewHandler(mock)

	req := httptest.NewRequest("GET", "/namespaces", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "+json") {
		t.Errorf("expected JSON content type, got %s", contentType)
	}
}

func TestContentNegotiationYAML(t *testing.T) {
	mock := &mockRegistry{}
	handler := NewHandler(mock)

	req := httptest.NewRequest("GET", "/namespaces", nil)
	req.Header.Set("Accept", "application/yaml")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "+yaml") {
		t.Errorf("expected YAML content type, got %s", contentType)
	}
}

func TestInvalidContentType(t *testing.T) {
	mock := &mockRegistry{}
	handler := NewHandler(mock)

	body := `{"name":"test"}`
	req := httptest.NewRequest("POST", "/namespaces", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestNotFound(t *testing.T) {
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

func TestInternalServerError(t *testing.T) {
	mock := &mockRegistry{
		readNamespaceFn: func(ctx context.Context, namespace string) (*registry.Namespace, error) {
			return nil, &registry.Error{
				Code:    registry.ErrorCodeInternalError,
				Message: "internal error",
			}
		},
	}

	handler := NewHandler(mock)
	req := httptest.NewRequest("GET", "/namespaces/test", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}
