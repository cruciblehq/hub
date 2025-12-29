package server

import (
	"io"
	"net/http"

	"github.com/cruciblehq/protocol/pkg/registry"
)

// Lists all channels for a resource.
//
// Returns a list of all channels associated with the specified resource, including
// their current version references and metadata.
func (h *Handler) listChannels(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue("namespace")
	resource := r.PathValue("resource")
	list, err := h.registry.ListChannels(r.Context(), namespace, resource)
	if err != nil {
		h.failWithError(w, r, err)
		return
	}
	h.encode(w, r, registry.MediaTypeChannelList, http.StatusOK, list)
}

// Creates a new channel.
//
// Channel names follow the same constraints as namespaces and resources.
// Returns an error if the channel already exists.
func (h *Handler) createChannel(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue("namespace")
	resource := r.PathValue("resource")
	var info registry.ChannelInfo
	if err := h.decode(r, registry.MediaTypeChannelInfo, &info); err != nil {
		h.fail(w, r, registry.ErrorCodeBadRequest, err.Error(), http.StatusBadRequest)
		return
	}

	ch, err := h.registry.CreateChannel(r.Context(), namespace, resource, info)
	if err != nil {
		h.failWithError(w, r, err)
		return
	}
	h.encode(w, r, registry.MediaTypeChannel, http.StatusCreated, ch)
}

// Updates an existing channel.
//
// Updates the version reference and metadata for the channel.
// Returns an error if the channel does not exist.
func (h *Handler) updateChannel(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue("namespace")
	resource := r.PathValue("resource")
	channel := r.PathValue("channel")
	var info registry.ChannelInfo
	if err := h.decode(r, registry.MediaTypeChannelInfo, &info); err != nil {
		h.fail(w, r, registry.ErrorCodeBadRequest, err.Error(), http.StatusBadRequest)
		return
	}

	ch, err := h.registry.UpdateChannel(r.Context(), namespace, resource, channel, info)
	if err != nil {
		h.failWithError(w, r, err)
		return
	}
	h.encode(w, r, registry.MediaTypeChannel, http.StatusOK, ch)
}

// Retrieves a specific channel.
//
// Returns channel metadata including its current version reference. Returns an
// error if the namespace, resource, or channel does not exist.
func (h *Handler) readChannel(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue("namespace")
	resource := r.PathValue("resource")
	channel := r.PathValue("channel")
	ch, err := h.registry.ReadChannel(r.Context(), namespace, resource, channel)
	if err != nil {
		h.failWithError(w, r, err)
		return
	}
	h.encode(w, r, registry.MediaTypeChannel, http.StatusOK, ch)
}

// Permanently deletes a channel.
//
// The operation is idempotent and succeeds if the channel does not exist.
// Deleting a channel does not affect the underlying versions it references.
func (h *Handler) deleteChannel(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue("namespace")
	resource := r.PathValue("resource")
	channel := r.PathValue("channel")
	if err := h.registry.DeleteChannel(r.Context(), namespace, resource, channel); err != nil {
		h.failWithError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Downloads the archive for a channel.
//
// Streams the compressed archive corresponding to the version currently
// referenced by the channel. Returns an error if the channel or its archive
// does not exist.
func (h *Handler) downloadChannelArchive(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue("namespace")
	resource := r.PathValue("resource")
	channel := r.PathValue("channel")

	// Read channel to get version
	ch, err := h.registry.ReadChannel(r.Context(), namespace, resource, channel)
	if err != nil {
		h.failWithError(w, r, err)
		return
	}

	// Download archive for that version
	archive, err := h.registry.DownloadArchive(r.Context(), namespace, resource, ch.Version.String)
	if err != nil {
		h.failWithError(w, r, err)
		return
	}
	defer archive.Close()

	w.Header().Set("Content-Type", string(registry.MediaTypeArchive))
	w.Header().Set("Content-Disposition", "attachment; filename=\""+resource+"-"+channel+".tar.zst\"")
	io.Copy(w, archive)
}
