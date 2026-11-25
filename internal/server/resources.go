package server

import (
	"net/http"

	"github.com/cruciblehq/spec/registry"
	"github.com/cruciblehq/utils-go/httpx"
)

// Lists all resources in a namespace.
//
// Returns a list of all resources within the specified namespace. The list
// order is implementation-dependent and may be empty if the namespace contains
// no resources.
func (h *Handler) listResources(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue(registry.ParamNamespace)
	list, err := h.registry.ListResources(r.Context(), namespace)
	if err != nil {
		h.failWithError(w, r, err)
		return
	}
	httpx.Write(w, registry.MediaTypeResourceList, registry.BodyFormat, http.StatusOK, list)
}

// Creates a new resource.
//
// Resource names follow the same constraints as namespace names. Returns an
// error if a resource with the given name already exists in the namespace.
func (h *Handler) createResource(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue(registry.ParamNamespace)
	var info registry.ResourceInfo
	if err := httpx.DecodeRequest(r, registry.MediaTypeResourceInfo, registry.BodyFormat, &info); err != nil {
		h.fail(w, r, registry.ErrorCodeBadRequest, err.Error(), http.StatusBadRequest)
		return
	}

	res, err := h.registry.CreateResource(r.Context(), namespace, info)
	if err != nil {
		h.failWithError(w, r, err)
		return
	}

	path := registry.ResourcePath(namespace, res.Name)
	w.Header().Set("Location", path)
	httpx.Write(w, registry.MediaTypeResource, registry.BodyFormat, http.StatusCreated, res)
}

// Retrieves resource metadata with version summaries.
//
// Returns resource information along with lightweight summaries of all
// versions. Returns an error if the namespace or resource does not exist.
func (h *Handler) readResource(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue(registry.ParamNamespace)
	resource := r.PathValue(registry.ParamResource)
	res, err := h.registry.ReadResource(r.Context(), namespace, resource)
	if err != nil {
		h.failWithError(w, r, err)
		return
	}
	httpx.Write(w, registry.MediaTypeResource, registry.BodyFormat, http.StatusOK, res)
}

// Updates mutable resource metadata.
//
// Immutable identifiers cannot be changed. Returns an error if the namespace or
// resource does not exist.
func (h *Handler) updateResource(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue(registry.ParamNamespace)
	resource := r.PathValue(registry.ParamResource)
	var info registry.ResourceInfo
	if err := httpx.DecodeRequest(r, registry.MediaTypeResourceInfo, registry.BodyFormat, &info); err != nil {
		h.fail(w, r, registry.ErrorCodeBadRequest, err.Error(), http.StatusBadRequest)
		return
	}

	res, err := h.registry.UpdateResource(r.Context(), namespace, resource, info)
	if err != nil {
		h.failWithError(w, r, err)
		return
	}
	httpx.Write(w, registry.MediaTypeResource, registry.BodyFormat, http.StatusOK, res)
}

// Permanently deletes a resource.
//
// Resources cannot be deleted if they contain any published versions. The
// operation is idempotent and succeeds if the resource does not exist.
func (h *Handler) deleteResource(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue(registry.ParamNamespace)
	resource := r.PathValue(registry.ParamResource)
	if err := h.registry.DeleteResource(r.Context(), namespace, resource); err != nil {
		h.failWithError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
