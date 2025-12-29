package server

import (
	"net/http"

	"github.com/cruciblehq/protocol/pkg/registry"
)

// Implements HTTP endpoints for the registry API.
//
// The registry implements the API defined in the [registry.Registry] interface,
// exposing endpoints for managing namespaces, resources, versions, and channels.
// This handler routes incoming HTTP requests to the appropriate methods on the
// underlying registry implementation.
type Handler struct {
	mux      *http.ServeMux
	registry registry.Registry
}

// Creates a new HTTP handler for the registry.
//
// Takes a [registry.Registry] implementation to handle the underlying data
// operations and sets up routing for all API endpoints.
func NewHandler(reg registry.Registry) *Handler {
	h := &Handler{
		mux:      http.NewServeMux(),
		registry: reg,
	}

	// Namespace routes
	h.mux.HandleFunc("GET /namespaces", h.listNamespaces)
	h.mux.HandleFunc("POST /namespaces", h.createNamespace)
	h.mux.HandleFunc("GET /namespaces/{namespace}", h.readNamespace)
	h.mux.HandleFunc("PUT /namespaces/{namespace}", h.updateNamespace)
	h.mux.HandleFunc("DELETE /namespaces/{namespace}", h.deleteNamespace)

	// Resource routes
	h.mux.HandleFunc("GET /namespaces/{namespace}/resources", h.listResources)
	h.mux.HandleFunc("POST /namespaces/{namespace}/resources", h.createResource)
	h.mux.HandleFunc("GET /namespaces/{namespace}/resources/{resource}", h.readResource)
	h.mux.HandleFunc("PUT /namespaces/{namespace}/resources/{resource}", h.updateResource)
	h.mux.HandleFunc("DELETE /namespaces/{namespace}/resources/{resource}", h.deleteResource)

	// Version routes
	h.mux.HandleFunc("GET /namespaces/{namespace}/resources/{resource}/versions", h.listVersions)
	h.mux.HandleFunc("POST /namespaces/{namespace}/resources/{resource}/versions", h.createVersion)
	h.mux.HandleFunc("GET /namespaces/{namespace}/resources/{resource}/versions/{version}", h.readVersion)
	h.mux.HandleFunc("PUT /namespaces/{namespace}/resources/{resource}/versions/{version}", h.updateVersion)
	h.mux.HandleFunc("DELETE /namespaces/{namespace}/resources/{resource}/versions/{version}", h.deleteVersion)
	h.mux.HandleFunc("PUT /namespaces/{namespace}/resources/{resource}/versions/{version}/archive", h.uploadArchive)
	h.mux.HandleFunc("GET /namespaces/{namespace}/resources/{resource}/versions/{version}/archive", h.downloadArchive)

	// Channel routes
	h.mux.HandleFunc("GET /namespaces/{namespace}/resources/{resource}/channels", h.listChannels)
	h.mux.HandleFunc("POST /namespaces/{namespace}/resources/{resource}/channels", h.createChannel)
	h.mux.HandleFunc("GET /namespaces/{namespace}/resources/{resource}/channels/{channel}", h.readChannel)
	h.mux.HandleFunc("PUT /namespaces/{namespace}/resources/{resource}/channels/{channel}", h.updateChannel)
	h.mux.HandleFunc("DELETE /namespaces/{namespace}/resources/{resource}/channels/{channel}", h.deleteChannel)
	h.mux.HandleFunc("GET /namespaces/{namespace}/resources/{resource}/channels/{channel}/archive", h.downloadChannelArchive)

	return h
}

// Serves HTTP requests by routing them to the appropriate handler methods.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}
