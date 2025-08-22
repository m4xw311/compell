package tools

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// ExecuteCommandTool implements the tool for running OS commands.
type ExecuteCommandTool struct {
	allowedCommands []string
}

func (t *ExecuteCommandTool) Name() string { return "execute_command" }
func (t *ExecuteCommandTool) Description() string {
	return "Executes a shell command. Args: command (string)."
}

func (t *ExecuteCommandTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	command, ok := args["command"].(string)
	if !ok {
		return "", fmt.Errorf("missing or invalid 'command' argument")
	}

	allowed, err := isCommandAllowed(command, t.allowedCommands)
	if err != nil {
		return "", err
	}
	if !allowed {
		return "", fmt.Errorf("command '%s' is not in the list of allowed commands", command)
	}

	// Basic shell-like execution
	parts := strings.Fields(command)
	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("command execution failed: %w. Output:\n%s", err, string(output))
	}

	return fmt.Sprintf("Command executed successfully. Output:\n%s", string(output)), nil
}
