package server

import (
	"fmt"
	"io"
	"net/http"

	"github.com/cruciblehq/spec/registry"
	"github.com/cruciblehq/utils-go/httpx"
)

// Lists all channels for a resource.
//
// Returns a list of all channels associated with the specified resource, including
// their current version references and metadata.
func (h *Handler) listChannels(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue(registry.ParamNamespace)
	resource := r.PathValue(registry.ParamResource)
	list, err := h.registry.ListChannels(r.Context(), namespace, resource)
	if err != nil {
		h.failWithError(w, r, err)
		return
	}
	httpx.Write(w, registry.MediaTypeChannelList, registry.BodyFormat, http.StatusOK, list)
}

// Creates a new channel.
//
// Channel names follow the same constraints as namespaces and resources.
// Returns an error if the channel already exists.
func (h *Handler) createChannel(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue(registry.ParamNamespace)
	resource := r.PathValue(registry.ParamResource)
	var info registry.ChannelInfo
	if err := httpx.DecodeRequest(r, registry.MediaTypeChannelInfo, registry.BodyFormat, &info); err != nil {
		h.fail(w, r, registry.ErrorCodeBadRequest, err.Error(), http.StatusBadRequest)
		return
	}

	ch, err := h.registry.CreateChannel(r.Context(), namespace, resource, info)
	if err != nil {
		h.failWithError(w, r, err)
		return
	}
	httpx.Write(w, registry.MediaTypeChannel, registry.BodyFormat, http.StatusCreated, ch)
}

// Updates an existing channel.
//
// Updates the version reference and metadata for the channel.
// Returns an error if the channel does not exist.
func (h *Handler) updateChannel(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue(registry.ParamNamespace)
	resource := r.PathValue(registry.ParamResource)
	channel := r.PathValue(registry.ParamChannel)
	var info registry.ChannelInfo
	if err := httpx.DecodeRequest(r, registry.MediaTypeChannelInfo, registry.BodyFormat, &info); err != nil {
		h.fail(w, r, registry.ErrorCodeBadRequest, err.Error(), http.StatusBadRequest)
		return
	}

	ch, err := h.registry.UpdateChannel(r.Context(), namespace, resource, channel, info)
	if err != nil {
		h.failWithError(w, r, err)
		return
	}
	httpx.Write(w, registry.MediaTypeChannel, registry.BodyFormat, http.StatusOK, ch)
}

// Retrieves a specific channel.
//
// Returns channel metadata including its current version reference. Returns an
// error if the namespace, resource, or channel does not exist.
func (h *Handler) readChannel(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue(registry.ParamNamespace)
	resource := r.PathValue(registry.ParamResource)
	channel := r.PathValue(registry.ParamChannel)
	ch, err := h.registry.ReadChannel(r.Context(), namespace, resource, channel)
	if err != nil {
		h.failWithError(w, r, err)
		return
	}
	httpx.Write(w, registry.MediaTypeChannel, registry.BodyFormat, http.StatusOK, ch)
}

// Permanently deletes a channel.
//
// The operation is idempotent and succeeds if the channel does not exist.
// Deleting a channel does not affect the underlying versions it references.
func (h *Handler) deleteChannel(w http.ResponseWriter, r *http.Request) {
	namespace := r.PathValue(registry.ParamNamespace)
	resource := r.PathValue(registry.ParamResource)
	channel := r.PathValue(registry.ParamChannel)
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
	namespace := r.PathValue(registry.ParamNamespace)
	resource := r.PathValue(registry.ParamResource)
	channel := r.PathValue(registry.ParamChannel)

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
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s-%s.tar.zst\"", resource, channel))
	io.Copy(w, archive)
}
