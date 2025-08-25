package tools

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/m4xw311/compell/errors"
)

// ExecuteCommandTool implements the tool for running OS commands.
type ExecuteCommandTool struct {
	allowedCommands []string
}

func (t *ExecuteCommandTool) Name() string { return "execute_command" }
func (t *ExecuteCommandTool) Description() string {
	if len(t.allowedCommands) == 0 {
		return "Executes a shell command. No commands are currently allowed. Args: command (string)."
	}

	allowedList := "Allowed command wildcard patterns:\n"
	for _, cmd := range t.allowedCommands {
		allowedList += fmt.Sprintf("- %s\n", cmd)
	}

	return fmt.Sprintf("Executes a shell command. Args: command (string).\n%s", allowedList)
}

func (t *ExecuteCommandTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	command, ok := args["command"].(string)
	if !ok {
		return "", errors.New("missing or invalid 'command' argument")
	}

	allowed, err := isCommandAllowed(command, t.allowedCommands)
	if err != nil {
		return "", err
	}
	if !allowed {
		return "", errors.New("command '%s' is not in the list of allowed commands", command)
	}

	// Basic shell-like execution
	parts := strings.Fields(command)
	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.Wrapf(err, "command execution failed. Output:\n%s", string(output))
	}

	return fmt.Sprintf("Command executed successfully. Output:\n%s", string(output)), nil
}
