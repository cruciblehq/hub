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

func TestListChannels(t *testing.T) {
	mock := &mockRegistry{
		listChannelsFn: func(ctx context.Context, namespace string, resource string) (*registry.ChannelList, error) {
			return &registry.ChannelList{
				Channels: []registry.ChannelSummary{
					{Name: "stable", Version: "1.0.0", Description: "Stable channel"},
				},
			}, nil
		},
	}

	handler := NewHandler(mock)
	req := httptest.NewRequest("GET", "/namespaces/test/resources/widget/channels", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "stable") {
		t.Errorf("expected response to contain channel name")
	}
}

func TestCreateChannel(t *testing.T) {
	mock := &mockRegistry{
		createChannelFn: func(ctx context.Context, namespace string, resource string, info registry.ChannelInfo) (*registry.Channel, error) {
			return &registry.Channel{
				Namespace:   namespace,
				Resource:    resource,
				Name:        info.Name,
				Version:     registry.Version{String: info.Version},
				Description: info.Description,
				CreatedAt:   1234567890,
			}, nil
		},
	}

	handler := NewHandler(mock)
	body := `{"name":"stable","version":"1.0.0","description":"Stable channel"}`
	req := httptest.NewRequest("POST", "/namespaces/test/resources/widget/channels", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/vnd.crucible.channel-info.v0+json")
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}
}

func TestUpdateChannel(t *testing.T) {
	mock := &mockRegistry{
		updateChannelFn: func(ctx context.Context, namespace string, resource string, channel string, info registry.ChannelInfo) (*registry.Channel, error) {
			return &registry.Channel{
				Namespace:   namespace,
				Resource:    resource,
				Name:        channel,
				Version:     registry.Version{String: info.Version},
				Description: info.Description,
				UpdatedAt:   1234567890,
			}, nil
		},
	}

	handler := NewHandler(mock)
	body := `{"name":"stable","version":"1.0.1","description":"Updated stable channel"}`
	req := httptest.NewRequest("PUT", "/namespaces/test/resources/widget/channels/stable", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/vnd.crucible.channel-info.v0+json")
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestReadChannel(t *testing.T) {
	mock := &mockRegistry{
		readChannelFn: func(ctx context.Context, namespace string, resource string, channel string) (*registry.Channel, error) {
			return &registry.Channel{
				Namespace:   namespace,
				Resource:    resource,
				Name:        channel,
				Version:     registry.Version{String: "1.0.0"},
				Description: "Stable channel",
				CreatedAt:   1234567890,
			}, nil
		},
	}

	handler := NewHandler(mock)
	req := httptest.NewRequest("GET", "/namespaces/test/resources/widget/channels/stable", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "stable") {
		t.Errorf("expected response to contain channel name")
	}
}

func TestDeleteChannel(t *testing.T) {
	mock := &mockRegistry{
		deleteChannelFn: func(ctx context.Context, namespace string, resource string, channel string) error {
			return nil
		},
	}

	handler := NewHandler(mock)
	req := httptest.NewRequest("DELETE", "/namespaces/test/resources/widget/channels/stable", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", w.Code)
	}
}

func TestDownloadChannelArchive(t *testing.T) {
	mock := &mockRegistry{
		readChannelFn: func(ctx context.Context, namespace string, resource string, channel string) (*registry.Channel, error) {
			return &registry.Channel{
				Namespace: namespace,
				Resource:  resource,
				Name:      channel,
				Version:   registry.Version{String: "1.0.0"},
			}, nil
		},
		downloadArchiveFn: func(ctx context.Context, namespace string, resource string, version string) (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader("channel archive data")), nil
		},
	}

	handler := NewHandler(mock)
	req := httptest.NewRequest("GET", "/namespaces/test/resources/widget/channels/stable/archive", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != string(registry.MediaTypeArchive) {
		t.Errorf("expected Content-Type %s, got %s", registry.MediaTypeArchive, contentType)
	}

	if w.Body.String() != "channel archive data" {
		t.Errorf("expected channel archive data in response body")
	}
}
