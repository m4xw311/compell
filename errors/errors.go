package errors

import (
	"fmt"
	"path/filepath"
	"runtime"
)

// New creates a new error with file and line number information.
func New(format string, a ...interface{}) error {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "???"
		line = 0
	} else {
		file = filepath.Base(file)
	}
	return fmt.Errorf("[%s:%d] %s", file, line, fmt.Sprintf(format, a...))
}

// Wrapf adds context (including file and line number) to an existing error.
// If the provided error is nil, Wrapf returns nil.
func Wrapf(err error, format string, a ...interface{}) error {
	if err == nil {
		return nil
	}
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "???"
		line = 0
	} else {
		file = filepath.Base(file)
	}
	return fmt.Errorf("[%s:%d] %s: %w", file, line, fmt.Sprintf(format, a...), err)
}
