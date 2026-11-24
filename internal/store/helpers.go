package store

import (
	"log/slog"

	"github.com/cruciblehq/spec/registry"
)

// Logs an error with structured context and returns a registry error.
//
// The error is logged with the provided message and key-value pairs, while the
// returned Error contains only the error code and message for the client. This
// prevents leaking internal implementation details.
func (s *SQLStore) logAndReturnError(code registry.ErrorCode, message string, err error, keyvals ...any) *registry.Error {
	args := make([]any, 0, 2+len(keyvals))
	args = append(args, "error", err)
	args = append(args, keyvals...)
	s.logger.Error(message, args...)
	return &registry.Error{
		Code:    code,
		Message: message,
	}
}

// Returns the default logger when none is provided.
func defaultLogger(logger *slog.Logger) *slog.Logger {
	if logger == nil {
		return slog.Default()
	}
	return logger
}
