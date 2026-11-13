// Package logging configures structured application logs.
package logging

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"resume-editor/internal/apperr"
)

// Open appends JSON records to a daily file. It never receives resume content.
func Open(dir, name, version string) (*slog.Logger, io.Closer, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, nil, apperr.Wrap(apperr.LogInit, err)
	}
	file, err := os.OpenFile(filepath.Join(dir, time.Now().Format("2006-01-02")+".jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, nil, apperr.Wrap(apperr.LogInit, err)
	}
	// A desktop app launched without a terminal may have no usable stderr.
	// Persist the record before attempting the optional console output.
	logger := slog.New(slog.NewJSONHandler(io.MultiWriter(file, os.Stderr), nil)).With("app", name, "version", version)
	return logger, file, nil
}

// Failure consistently records the public code and the underlying diagnostic.
func Failure(logger *slog.Logger, operation string, err error) {
	appErr := apperr.Wrap(apperr.Unexpected, err)
	attrs := []any{"operation", operation, "code", appErr.Code}
	if cause := appErr.Unwrap(); cause != nil {
		attrs = append(attrs, "cause", cause.Error())
	}
	logger.Error("operation_failed", attrs...)
}
