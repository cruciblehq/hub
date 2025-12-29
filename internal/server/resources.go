package server

import (
	"net/http"
	"net/url"

	"github.com/cruciblehq/protocol/pkg/registry"
)

// Lists all resources in a namespace.
//
// Returns a list of all resources within the specified namespace. The list
// order is implementation-dependent and may be empty if the namespace contains
// no resources.
func (h *Handler) listResources(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue("namespace")
	list, err := h.registry.ListResources(r.Context(), namespace)
	if err != nil {
		h.failWithError(w, r, err)
		return
	}
	h.encode(w, r, registry.MediaTypeResourceList, http.StatusOK, list)
}

// Creates a new resource.
//
// Resource names follow the same constraints as namespace names. Returns an
// error if a resource with the given name already exists in the namespace.
func (h *Handler) createResource(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue("namespace")
	var info registry.ResourceInfo
	if err := h.decode(r, registry.MediaTypeResourceInfo, &info); err != nil {
		h.fail(w, r, registry.ErrorCodeBadRequest, err.Error(), http.StatusBadRequest)
		return
	}

	res, err := h.registry.CreateResource(r.Context(), namespace, info)
	if err != nil {
		h.failWithError(w, r, err)
		return
	}

	path, _ := url.JoinPath("/namespaces", namespace, "resources", res.Name)
	w.Header().Set("Location", path)
	h.encode(w, r, registry.MediaTypeResource, http.StatusCreated, res)
}

// Retrieves resource metadata with version and channel summaries.
//
// Returns resource information along with lightweight summaries of all versions
// and channels. Returns an error if the namespace or resource does not exist.
func (h *Handler) readResource(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue("namespace")
	resource := r.PathValue("resource")
	res, err := h.registry.ReadResource(r.Context(), namespace, resource)
	if err != nil {
		h.failWithError(w, r, err)
		return
	}
	h.encode(w, r, registry.MediaTypeResource, http.StatusOK, res)
}

// Updates mutable resource metadata.
//
// Immutable identifiers cannot be changed. Returns an error if the namespace or
// resource does not exist.
func (h *Handler) updateResource(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue("namespace")
	resource := r.PathValue("resource")
	var info registry.ResourceInfo
	if err := h.decode(r, registry.MediaTypeResourceInfo, &info); err != nil {
		h.fail(w, r, registry.ErrorCodeBadRequest, err.Error(), http.StatusBadRequest)
		return
	}

	res, err := h.registry.UpdateResource(r.Context(), namespace, resource, info)
	if err != nil {
		h.failWithError(w, r, err)
		return
	}
	h.encode(w, r, registry.MediaTypeResource, http.StatusOK, res)
}

// Permanently deletes a resource.
//
// Resources cannot be deleted if they contain any published versions. The
// operation is idempotent and succeeds if the resource does not exist.
func (h *Handler) deleteResource(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue("namespace")
	resource := r.PathValue("resource")
	if err := h.registry.DeleteResource(r.Context(), namespace, resource); err != nil {
		h.failWithError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
