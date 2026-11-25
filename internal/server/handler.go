package server

import (
	"fmt"
	"net/http"

	"github.com/cruciblehq/spec/registry"
)

// Implements HTTP endpoints for the registry API.
//
// The registry implements the API defined in the [registry.Registry] interface,
// exposing endpoints for managing namespaces, resources, and versions. This
// handler routes incoming HTTP requests to the appropriate methods on the
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
	h.mux.HandleFunc(fmt.Sprintf("GET %s", registry.PatternNamespaceList), h.listNamespaces)
	h.mux.HandleFunc(fmt.Sprintf("POST %s", registry.PatternNamespaceList), h.createNamespace)
	h.mux.HandleFunc(fmt.Sprintf("GET %s", registry.PatternNamespace), h.readNamespace)
	h.mux.HandleFunc(fmt.Sprintf("PUT %s", registry.PatternNamespace), h.updateNamespace)
	h.mux.HandleFunc(fmt.Sprintf("DELETE %s", registry.PatternNamespace), h.deleteNamespace)

	// Resource routes
	h.mux.HandleFunc(fmt.Sprintf("GET %s", registry.PatternResourceList), h.listResources)
	h.mux.HandleFunc(fmt.Sprintf("POST %s", registry.PatternResourceList), h.createResource)
	h.mux.HandleFunc(fmt.Sprintf("GET %s", registry.PatternResource), h.readResource)
	h.mux.HandleFunc(fmt.Sprintf("PUT %s", registry.PatternResource), h.updateResource)
	h.mux.HandleFunc(fmt.Sprintf("DELETE %s", registry.PatternResource), h.deleteResource)

	// Version routes
	h.mux.HandleFunc(fmt.Sprintf("GET %s", registry.PatternVersionList), h.listVersions)
	h.mux.HandleFunc(fmt.Sprintf("POST %s", registry.PatternVersionList), h.createVersion)
	h.mux.HandleFunc(fmt.Sprintf("GET %s", registry.PatternVersion), h.readVersion)
	h.mux.HandleFunc(fmt.Sprintf("PUT %s", registry.PatternVersion), h.updateVersion)
	h.mux.HandleFunc(fmt.Sprintf("DELETE %s", registry.PatternVersion), h.deleteVersion)
	h.mux.HandleFunc(fmt.Sprintf("PUT %s", registry.PatternArchive), h.uploadArchive)
	h.mux.HandleFunc(fmt.Sprintf("GET %s", registry.PatternArchive), h.downloadArchive)

	return h
}

// Serves HTTP requests by routing them to the appropriate handler methods.
//
// Requests without a valid credential are refused before routing.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !h.authenticated(r) {
		h.fail(w, r, registry.ErrorCodeUnauthorized, "the request has no valid credential", http.StatusUnauthorized)
		return
	}
	h.mux.ServeHTTP(w, r)
}

// Whether the request carries a valid credential.
//
// Placeholder that accepts every request. Sentinel, Crucible's authentication
// service, will check the bearer token here.
func (h *Handler) authenticated(_ *http.Request) bool {
	return true
}
