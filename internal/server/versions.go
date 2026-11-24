package server

import (
	"fmt"
	"io"
	"net/http"

	"github.com/cruciblehq/spec/registry"
	"github.com/cruciblehq/utils-go/httpx"
)

// Lists all versions of a resource.
//
// Returns a list of all versions within the specified resource. Version order
// is implementation-dependent and the list may be empty.
func (h *Handler) listVersions(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue(registry.ParamNamespace)
	resource := r.PathValue(registry.ParamResource)
	list, err := h.registry.ListVersions(r.Context(), namespace, resource)
	if err != nil {
		h.failWithError(w, r, err)
		return
	}
	httpx.Write(w, registry.MediaTypeVersionList, registry.BodyFormat, http.StatusOK, list)
}

// Creates a new version.
//
// Version strings must follow semantic versioning conventions. Versions are
// created in an unpublished state without an associated archive. Returns an
// error if the version already exists or the version string is invalid.
func (h *Handler) createVersion(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue(registry.ParamNamespace)
	resource := r.PathValue(registry.ParamResource)
	var info registry.VersionInfo
	if err := httpx.DecodeRequest(r, registry.MediaTypeVersionInfo, registry.BodyFormat, &info); err != nil {
		h.fail(w, r, registry.ErrorCodeBadRequest, err.Error(), http.StatusBadRequest)
		return
	}

	ver, err := h.registry.CreateVersion(r.Context(), namespace, resource, info)
	if err != nil {
		h.failWithError(w, r, err)
		return
	}

	path := registry.VersionPath(namespace, resource, ver.String)
	w.Header().Set("Location", path)
	httpx.Write(w, registry.MediaTypeVersion, registry.BodyFormat, http.StatusCreated, ver)
}

// Retrieves version metadata.
//
// Returns complete version information including archive details if uploaded.
// Returns an error if the namespace, resource, or version does not exist.
func (h *Handler) readVersion(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue(registry.ParamNamespace)
	resource := r.PathValue(registry.ParamResource)
	version := r.PathValue(registry.ParamVersion)
	ver, err := h.registry.ReadVersion(r.Context(), namespace, resource, version)
	if err != nil {
		h.failWithError(w, r, err)
		return
	}
	httpx.Write(w, registry.MediaTypeVersion, registry.BodyFormat, http.StatusOK, ver)
}

// Updates a private version.
//
// Setting public in the body makes the version public, which cannot be undone.
// The version string itself cannot be changed. Returns an error if the version
// does not exist or is already public.
func (h *Handler) updateVersion(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue(registry.ParamNamespace)
	resource := r.PathValue(registry.ParamResource)
	version := r.PathValue(registry.ParamVersion)
	var info registry.VersionInfo
	if err := httpx.DecodeRequest(r, registry.MediaTypeVersionInfo, registry.BodyFormat, &info); err != nil {
		h.fail(w, r, registry.ErrorCodeBadRequest, err.Error(), http.StatusBadRequest)
		return
	}

	ver, err := h.registry.UpdateVersion(r.Context(), namespace, resource, version, info)
	if err != nil {
		h.failWithError(w, r, err)
		return
	}
	httpx.Write(w, registry.MediaTypeVersion, registry.BodyFormat, http.StatusOK, ver)
}

// Permanently deletes a version.
//
// The operation is idempotent and succeeds if the version does not exist.
func (h *Handler) deleteVersion(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue(registry.ParamNamespace)
	resource := r.PathValue(registry.ParamResource)
	version := r.PathValue(registry.ParamVersion)
	if err := h.registry.DeleteVersion(r.Context(), namespace, resource, version); err != nil {
		h.failWithError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Uploads an archive for a version.
//
// Associates a compressed archive with a version. A private version's archive
// is replaced by uploading again; a public version's archive cannot change.
// The Archive-Digest header must contain the archive's cryptographic digest.
func (h *Handler) uploadArchive(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue(registry.ParamNamespace)
	resource := r.PathValue(registry.ParamResource)
	version := r.PathValue(registry.ParamVersion)

	ver, err := h.registry.UploadArchive(r.Context(), namespace, resource, version, r.Body)
	if err != nil {
		h.failWithError(w, r, err)
		return
	}
	httpx.Write(w, registry.MediaTypeVersion, registry.BodyFormat, http.StatusOK, ver)
}

// Downloads an archive for a version.
//
// Streams the compressed archive corresponding to the specified version.
// Returns an error if the version or its archive does not exist.
func (h *Handler) downloadArchive(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue(registry.ParamNamespace)
	resource := r.PathValue(registry.ParamResource)
	version := r.PathValue(registry.ParamVersion)

	archive, err := h.registry.DownloadArchive(r.Context(), namespace, resource, version)
	if err != nil {
		h.failWithError(w, r, err)
		return
	}
	defer archive.Close()

	w.Header().Set("Content-Type", string(registry.MediaTypeArchive))
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s-%s.tar.zst\"", resource, version))
	io.Copy(w, archive)
}
