package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/cruciblehq/protocol/pkg/codec"
	"github.com/cruciblehq/protocol/pkg/registry"
)

// Decodes the request body.
//
// The result is decoded into the provided value (v) after validating the
// Content-Type header. Validates that the Content-Type matches the expected
// media type, returning an error if the Content-Type doesn't match or the
// format is unsupported.
func (h *Handler) decode(r *http.Request, expected registry.MediaType, v interface{}) error {
	header := r.Header.Get("Content-Type")

	// Parse content type
	contentType, mediaType, err := codec.Parse(header)
	if err != nil {
		return fmt.Errorf("invalid Content-Type: %w", err)
	}

	// Validate base media type matches expected
	if !strings.EqualFold(mediaType, string(expected)) {
		return fmt.Errorf("expected Content-Type %s+{format}, got %s", expected, header)
	}

	// Decode
	return codec.Decode(r.Body, contentType, "field", v)
}

// Encodes and writes a response with the specified media type and status code.
//
// Sets the Content-Type header to the specified media type with an encoding
// suffix, writes the status code, and encodes the provided value in the body.
// The format is negotiated based on the Accept header.
func (h *Handler) encode(w http.ResponseWriter, r *http.Request, mediaType registry.MediaType, status int, v interface{}) error {
	format := codec.Negotiate(r.Header.Get("Accept"))

	w.Header().Set("Content-Type", string(mediaType)+format.Suffix())
	w.WriteHeader(status)

	// Encode
	return codec.Encode(w, format, "field", v)
}

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
	h.encode(w, r, registry.MediaTypeError, status, err)
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
		registry.ErrorCodeVersionExists, registry.ErrorCodeChannelExists,
		registry.ErrorCodeNamespaceNotEmpty, registry.ErrorCodeResourceHasPublished,
		registry.ErrorCodeVersionPublished:
		return http.StatusConflict
	case registry.ErrorCodePreconditionFailed:
		return http.StatusPreconditionFailed
	case registry.ErrorCodeUnsupportedMediaType:
		return http.StatusUnsupportedMediaType
	case registry.ErrorCodeNotAcceptable:
		return http.StatusNotAcceptable
	default:
		return http.StatusInternalServerError
	}
}
