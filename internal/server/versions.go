package server

import (
	"io"
	"net/http"
	"net/url"

	"github.com/cruciblehq/protocol/pkg/registry"
)

// Lists all versions of a resource.
//
// Returns a list of all versions within the specified resource. Version order
// is implementation-dependent and the list may be empty.
func (h *Handler) listVersions(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue("namespace")
	resource := r.PathValue("resource")
	list, err := h.registry.ListVersions(r.Context(), namespace, resource)
	if err != nil {
		h.failWithError(w, r, err)
		return
	}
	h.encode(w, r, registry.MediaTypeVersionList, http.StatusOK, list)
}

// Creates a new version.
//
// Version strings must follow semantic versioning conventions. Versions are
// created in an unpublished state without an associated archive. Returns an
// error if the version already exists or the version string is invalid.
func (h *Handler) createVersion(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue("namespace")
	resource := r.PathValue("resource")
	var info registry.VersionInfo
	if err := h.decode(r, registry.MediaTypeVersionInfo, &info); err != nil {
		h.fail(w, r, registry.ErrorCodeBadRequest, err.Error(), http.StatusBadRequest)
		return
	}

	ver, err := h.registry.CreateVersion(r.Context(), namespace, resource, info)
	if err != nil {
		h.failWithError(w, r, err)
		return
	}

	path, _ := url.JoinPath("/namespaces", namespace, "resources", resource, "versions", ver.String)
	w.Header().Set("Location", path)
	h.encode(w, r, registry.MediaTypeVersion, http.StatusCreated, ver)
}

// Retrieves version metadata.
//
// Returns complete version information including archive details if uploaded.
// Returns an error if the namespace, resource, or version does not exist.
func (h *Handler) readVersion(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue("namespace")
	resource := r.PathValue("resource")
	version := r.PathValue("version")
	ver, err := h.registry.ReadVersion(r.Context(), namespace, resource, version)
	if err != nil {
		h.failWithError(w, r, err)
		return
	}
	h.encode(w, r, registry.MediaTypeVersion, http.StatusOK, ver)
}

// Updates mutable version metadata.
//
// Only unpublished versions can be updated. The version string itself cannot be
// changed. Returns an error if the version does not exist or is already published.
func (h *Handler) updateVersion(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue("namespace")
	resource := r.PathValue("resource")
	version := r.PathValue("version")
	var info registry.VersionInfo
	if err := h.decode(r, registry.MediaTypeVersionInfo, &info); err != nil {
		h.fail(w, r, registry.ErrorCodeBadRequest, err.Error(), http.StatusBadRequest)
		return
	}

	ver, err := h.registry.UpdateVersion(r.Context(), namespace, resource, version, info)
	if err != nil {
		h.failWithError(w, r, err)
		return
	}
	h.encode(w, r, registry.MediaTypeVersion, http.StatusOK, ver)
}

// Permanently deletes a version.
//
// Only unpublished versions can be deleted. The operation is idempotent and
// succeeds if the version does not exist.
func (h *Handler) deleteVersion(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue("namespace")
	resource := r.PathValue("resource")
	version := r.PathValue("version")
	if err := h.registry.DeleteVersion(r.Context(), namespace, resource, version); err != nil {
		h.failWithError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Uploads an archive for a version.
//
// Associates a compressed archive with a version. The archive can be replaced
// by uploading again. Publishing is a separate operation. The Archive-Digest
// header must contain the archive's cryptographic digest.
func (h *Handler) uploadArchive(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue("namespace")
	resource := r.PathValue("resource")
	version := r.PathValue("version")

	ver, err := h.registry.UploadArchive(r.Context(), namespace, resource, version, r.Body)
	if err != nil {
		h.failWithError(w, r, err)
		return
	}
	h.encode(w, r, registry.MediaTypeVersion, http.StatusOK, ver)
}

// Downloads an archive for a version.
//
// Streams the compressed archive corresponding to the specified version.
// Returns an error if the version or its archive does not exist.
func (h *Handler) downloadArchive(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue("namespace")
	resource := r.PathValue("resource")
	version := r.PathValue("version")

	archive, err := h.registry.DownloadArchive(r.Context(), namespace, resource, version)
	if err != nil {
		h.failWithError(w, r, err)
		return
	}
	defer archive.Close()

	w.Header().Set("Content-Type", string(registry.MediaTypeArchive))
	w.Header().Set("Content-Disposition", "attachment; filename=\""+resource+"-"+version+".tar.zst\"")
	io.Copy(w, archive)
}
