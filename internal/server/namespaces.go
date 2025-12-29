package server

import (
	"net/http"
	"net/url"

	"github.com/cruciblehq/protocol/pkg/registry"
)

// Lists all namespaces.
//
// Returns a list of all existing namespaces in the registry. The list order is
// implementation-dependent and may be empty if no namespaces exist.
func (h *Handler) listNamespaces(w http.ResponseWriter, r *http.Request) {
	list, err := h.registry.ListNamespaces(r.Context())
	if err != nil {
		h.failWithError(w, r, err)
		return
	}
	h.encode(w, r, registry.MediaTypeNamespaceList, http.StatusOK, list)
}

// Creates a new namespace.
//
// Namespace names may include lowercase letters (a–z), digits (0–9), and
// hyphens (-), must start and end with an alphanumeric character, and must not
// exceed 63 characters. Returns an error if a namespace with the given name
// already exists.
func (h *Handler) createNamespace(w http.ResponseWriter, r *http.Request) {
	var info registry.NamespaceInfo
	if err := h.decode(r, registry.MediaTypeNamespaceInfo, &info); err != nil {
		h.fail(w, r, registry.ErrorCodeBadRequest, err.Error(), http.StatusBadRequest)
		return
	}

	ns, err := h.registry.CreateNamespace(r.Context(), info)
	if err != nil {
		h.failWithError(w, r, err)
		return
	}

	path, _ := url.JoinPath("/namespaces", ns.Name)
	w.Header().Set("Location", path)
	h.encode(w, r, registry.MediaTypeNamespace, http.StatusCreated, ns)
}

// Retrieves namespace metadata and resource summaries.
//
// Returns namespace information along with lightweight summaries of all
// contained resources. Returns an error if the namespace does not exist.
func (h *Handler) readNamespace(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue("namespace")
	ns, err := h.registry.ReadNamespace(r.Context(), namespace)
	if err != nil {
		h.failWithError(w, r, err)
		return
	}
	h.encode(w, r, registry.MediaTypeNamespace, http.StatusOK, ns)
}

// Updates mutable namespace metadata.
//
// Immutable identifiers cannot be changed. Updating metadata does not affect
// contained resources or their timestamps. Returns an error if the namespace
// does not exist.
func (h *Handler) updateNamespace(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue("namespace")
	var info registry.NamespaceInfo
	if err := h.decode(r, registry.MediaTypeNamespaceInfo, &info); err != nil {
		h.fail(w, r, registry.ErrorCodeBadRequest, err.Error(), http.StatusBadRequest)
		return
	}

	ns, err := h.registry.UpdateNamespace(r.Context(), namespace, info)
	if err != nil {
		h.failWithError(w, r, err)
		return
	}
	h.encode(w, r, registry.MediaTypeNamespace, http.StatusOK, ns)
}

// Permanently deletes a namespace.
//
// Namespaces cannot be deleted if they contain any resources. The operation is
// idempotent and succeeds if the namespace does not exist.
func (h *Handler) deleteNamespace(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue("namespace")
	if err := h.registry.DeleteNamespace(r.Context(), namespace); err != nil {
		h.failWithError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
