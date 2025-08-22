package tools

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/m4xw311/compell/config"
)

// Tool defines the interface for any action the agent can take.
type Tool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, args map[string]interface{}) (string, error)
}

// ToolRegistry holds all available tools.
type ToolRegistry struct {
	tools map[string]Tool
}

func NewToolRegistry(cfg *config.Config) *ToolRegistry {
	r := &ToolRegistry{tools: make(map[string]Tool)}

	// Register default tools
	r.Register(&ReadFileTool{fsAccess: &cfg.FilesystemAccess})
	r.Register(&WriteFileTool{fsAccess: &cfg.FilesystemAccess})
	r.Register(&ExecuteCommandTool{allowedCommands: cfg.AllowedCommands})
	// Add other tools like CreateDir, DeleteFile, ReadRepo here...

	// Placeholder for registering MCP tools
	for _, mcpServer := range cfg.AdditionalMCPServers {
		fmt.Printf("INFO: MCP Server '%s' would be initialized here.\n", mcpServer.Name)
		// Here you would start the subprocess and create an MCPTool that
		// communicates with it.
	}

	return r
}

func (r *ToolRegistry) Register(t Tool) {
	r.tools[t.Name()] = t
}

func (r *ToolRegistry) GetTool(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

// GetActiveTools returns the tool instances for a given toolset.
func (r *ToolRegistry) GetActiveTools(ts *config.Toolset) ([]Tool, error) {
	var activeTools []Tool
	for _, toolName := range ts.Tools {
		// Handle MCP tools like <server>:<tool>
		if strings.Contains(toolName, ":") {
			// This is where you would get your MCP-specific tool
			fmt.Printf("NOTICE: MCP tool '%s' not yet implemented.\n", toolName)
			continue
		}

		if t, ok := r.GetTool(toolName); ok {
			activeTools = append(activeTools, t)
		} else {
			return nil, fmt.Errorf("tool '%s' from toolset '%s' is not registered", toolName, ts.Name)
		}
	}
	return activeTools, nil
}

// isPathRestricted checks if a path matches any of the glob patterns.
func isPathRestricted(path string, patterns []string) (bool, error) {
	for _, pattern := range patterns {
		match, err := doublestar.PathMatch(pattern, path)
		if err != nil {
			return false, fmt.Errorf("invalid glob pattern '%s': %w", pattern, err)
		}
		if match {
			return true, nil
		}
	}
	return false, nil
}

// isCommandAllowed checks if a command is in the allowlist (with regex support).
func isCommandAllowed(command string, allowed []string) (bool, error) {
	cmdParts := strings.Fields(command)
	if len(cmdParts) == 0 {
		return false, nil
	}

	for _, pattern := range allowed {
		re, err := regexp.Compile(pattern)
		if err != nil {
			fmt.Printf("Warning: Invalid regex in allowed_commands '%s': %v\n", pattern, err)
			// Fallback to simple string comparison if regex is invalid
			if command == pattern {
				return true, nil
			}
			continue
		}
		if re.MatchString(command) {
			return true, nil
		}
	}
	return false, nil
}
