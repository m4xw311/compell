package tools

import (
	"context"
	"fmt"
	"os"
	"strings"

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
	return "Writes content to a file. Overwrites the file unless optional `start_line` and `end_line` are provided to replace a specific range. Args: path (string), content (string), [start_line (int)], [end_line (int)]."
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

	startLineRaw, startOk := args["start_line"]
	endLineRaw, endOk := args["end_line"]

	// If start and end lines are provided, perform partial replacement.
	if startOk || endOk {
		// Both must be provided for a partial write.
		if !(startOk && endOk) {
			return "", errors.New("for partial write, both 'start_line' and 'end_line' must be provided")
		}

		start, ok := startLineRaw.(float64)
		if !ok {
			return "", errors.New("invalid 'start_line' argument: must be a number")
		}
		end, ok := endLineRaw.(float64)
		if !ok {
			return "", errors.New("invalid 'end_line' argument: must be a number")
		}
		return t.executePartialWrite(path, content, int(start), int(end))
	}

	// Otherwise, perform a full overwrite.
	err = os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		return "", errors.Wrapf(err, "failed to write to file '%s'", path)
	}
	return fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path), nil
}

func (t *WriteFileTool) executePartialWrite(path, newContent string, startLine, endLine int) (string, error) {
	if startLine <= 0 || endLine < startLine {
		return "", errors.New("invalid line numbers: start_line must be >= 1 and end_line must be >= start_line")
	}

	fileBytes, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", errors.New("cannot perform partial write: file '%s' does not exist", path)
		}
		return "", errors.Wrapf(err, "failed to read file for partial write '%s'", path)
	}

	lines := strings.Split(string(fileBytes), "\n")

	if startLine > len(lines) {
		return "", errors.New("start_line %d is greater than the number of lines in the file (%d)", startLine, len(lines))
	}
	if endLine > len(lines) {
		return "", errors.New("end_line %d is greater than the number of lines in the file (%d)", endLine, len(lines))
	}

	// Rebuild the file content with the replacement.
	var newLines []string
	// Lines before the start (startLine is 1-based).
	newLines = append(newLines, lines[:startLine-1]...)
	// The new content.
	newLines = append(newLines, newContent)
	// Lines after the end.
	newLines = append(newLines, lines[endLine:]...)

	output := strings.Join(newLines, "\n")
	err = os.WriteFile(path, []byte(output), 0644)
	if err != nil {
		return "", errors.Wrapf(err, "failed to write updated content to file '%s'", path)
	}

	return fmt.Sprintf("Successfully replaced lines %d-%d in %s", startLine, endLine, path), nil
}

// CreateDirTool implements the tool for creating a directory.
type CreateDirTool struct {
	fsAccess *config.FilesystemAccess
}

func (t *CreateDirTool) Name() string { return "create_dir" }
func (t *CreateDirTool) Description() string {
	return "Creates a new directory. Args: path (string)."
}

func (t *CreateDirTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
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

	readOnly, err := isPathRestricted(path, t.fsAccess.ReadOnly)
	if err != nil {
		return "", err
	}
	if readOnly {
		return "", errors.New("access denied: path '%s' is read-only", path)
	}

	err = os.MkdirAll(path, 0755)
	if err != nil {
		return "", errors.Wrapf(err, "failed to create directory '%s'", path)
	}
	return fmt.Sprintf("Successfully created directory %s", path), nil
}

// DeleteFileTool implements the tool for deleting a file.
type DeleteFileTool struct {
	fsAccess *config.FilesystemAccess
}

func (t *DeleteFileTool) Name() string { return "delete_file" }
func (t *DeleteFileTool) Description() string {
	return "Deletes a file. Args: path (string)."
}

func (t *DeleteFileTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
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

	readOnly, err := isPathRestricted(path, t.fsAccess.ReadOnly)
	if err != nil {
		return "", err
	}
	if readOnly {
		return "", errors.New("access denied: path '%s' is read-only", path)
	}

	err = os.Remove(path)
	if err != nil {
		return "", errors.Wrapf(err, "failed to delete file '%s'", path)
	}
	return fmt.Sprintf("Successfully deleted file %s", path), nil
}

// DeleteDirTool implements the tool for deleting a directory.
type DeleteDirTool struct {
	fsAccess *config.FilesystemAccess
}

func (t *DeleteDirTool) Name() string { return "delete_dir" }
func (t *DeleteDirTool) Description() string {
	return "Deletes an empty directory. Args: path (string)."
}

func (t *DeleteDirTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
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

	readOnly, err := isPathRestricted(path, t.fsAccess.ReadOnly)
	if err != nil {
		return "", err
	}
	if readOnly {
		return "", errors.New("access denied: path '%s' is read-only", path)
	}

	// os.Remove will fail on a non-empty directory, which is the desired behavior.
	err = os.Remove(path)
	if err != nil {
		return "", errors.Wrapf(err, "failed to delete directory '%s'", path)
	}
	return fmt.Sprintf("Successfully deleted directory %s", path), nil
}
