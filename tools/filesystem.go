package tools

import (
	"context"
	"fmt"
	"os"

	"github.com/m4xw311/compell/config"
	"github.com/m4xw311/compell/errors"
)

// ReadFileTool implements the tool for reading a file.
type ReadFileTool struct {
	fsAccess *config.FilesystemAccess
}

func (t *ReadFileTool) Name() string { return "read_file" }
func (t *ReadFileTool) Description() string {
	return "Reads the entire content of a file. Args: path (string)."
}

func (t *ReadFileTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	path, ok := args["path"].(string)
	if !ok {
		return "", errors.New("missing or invalid 'path' argument")
	}

	hidden, err := isPathRestricted(path, t.fsAccess.Hidden)
	if err != nil {
		return "", err
	}
	if hidden {
		return "", errors.New("access denied: path '%s' is hidden", path)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return "", errors.Wrapf(err, "failed to read file '%s'", path)
	}
	return string(content), nil
}

// WriteFileTool implements the tool for writing to a file.
type WriteFileTool struct {
	fsAccess *config.FilesystemAccess
}

func (t *WriteFileTool) Name() string { return "write_file" }
func (t *WriteFileTool) Description() string {
	return "Writes content to a file, replacing it entirely. Args: path (string), content (string)."
}

func (t *WriteFileTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	path, pathOk := args["path"].(string)
	content, contentOk := args["content"].(string)
	if !pathOk || !contentOk {
		return "", errors.New("missing or invalid 'path' or 'content' arguments")
	}

	hidden, err := isPathRestricted(path, t.fsAccess.Hidden)
	if err != nil {
		return "", err
	}
	if hidden {
		return "", errors.New("access denied: path '%s' is hidden", path)
	}

	readOnly, err := isPathRestricted(path, t.fsAccess.ReadOnly)
	if err != nil {
		return "", err
	}
	if readOnly {
		return "", errors.New("access denied: path '%s' is read-only", path)
	}

	err = os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		return "", errors.Wrapf(err, "failed to write to file '%s'", path)
	}
	return fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path), nil
}
