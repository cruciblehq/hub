package server

import (
	"net/http"

	"github.com/cruciblehq/spec/registry"
	"github.com/cruciblehq/utils-go/httpx"
)

// Writes an error response.
//
// Constructs a [registry.Error] with the provided code and message, and
// encodes it with the specified HTTP status code. Then writes the response
// with the [registry.MediaTypeError] media type.
func (h *Handler) fail(w http.ResponseWriter, r *http.Request, code registry.ErrorCode, message string, status int) {
	err := &registry.Error{
		Code:    code,
		Message: message,
	}
	httpx.Write(w, registry.MediaTypeError, registry.BodyFormat, status, err)
}

// Handles errors by converting them to appropriate HTTP responses.
//
// Extracts [registry.Error] for proper status code mapping, defaulting to 500
// for other errors or unknown codes. Then writes the error response using the
// appropriate HTTP status code and media type.
func (h *Handler) failWithError(w http.ResponseWriter, r *http.Request, err error) {
	if regErr, ok := err.(*registry.Error); ok {
		status := h.errorCodeToHTTPStatus(regErr.Code)
		h.fail(w, r, regErr.Code, regErr.Message, status)
		return
	}

	// Default to internal server error
	h.fail(w, r, registry.ErrorCodeInternalError, err.Error(), http.StatusInternalServerError)
}

// Maps registry error codes to HTTP status codes.
//
// Provides appropriate HTTP status for each [registry.ErrorCode], defaulting
// to 500 Internal Server Error for unknown codes.
func (h *Handler) errorCodeToHTTPStatus(code registry.ErrorCode) int {
	switch code {
	case registry.ErrorCodeBadRequest:
		return http.StatusBadRequest
	case registry.ErrorCodeNotFound:
		return http.StatusNotFound
	case registry.ErrorCodeNamespaceExists, registry.ErrorCodeResourceExists,
		registry.ErrorCodeVersionExists, registry.ErrorCodeVersionPublic, registry.ErrorCodeVersionSuperseded, registry.ErrorCodeChannelExists,
		registry.ErrorCodeNamespaceNotEmpty, registry.ErrorCodeResourceHasPublished,
		registry.ErrorCodeVersionPublished:
		return http.StatusConflict
	case registry.ErrorCodePreconditionFailed:
		return http.StatusPreconditionFailed
	default:
		return http.StatusInternalServerError
	}
}
