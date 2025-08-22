package tools

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/m4xw311/compell/config"
	"github.com/m4xw311/compell/errors"
	"github.com/m4xw311/compell/tools/mcp"
)

// Tool defines the interface for any action the agent can take.
type Tool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, args map[string]interface{}) (string, error)
}

// ToolRegistry holds all available tools.
type ToolRegistry struct {
	tools      map[string]Tool
	mcpClients map[string]*mcp.MCPClient
}

func NewToolRegistry(cfg *config.Config) *ToolRegistry {
	r := &ToolRegistry{
		tools:      make(map[string]Tool),
		mcpClients: make(map[string]*mcp.MCPClient),
	}

	// Register default tools
	r.Register(&ReadFileTool{fsAccess: &cfg.FilesystemAccess})
	r.Register(&WriteFileTool{fsAccess: &cfg.FilesystemAccess})
	r.Register(&CreateDirTool{fsAccess: &cfg.FilesystemAccess})
	r.Register(&DeleteFileTool{fsAccess: &cfg.FilesystemAccess})
	r.Register(&DeleteDirTool{fsAccess: &cfg.FilesystemAccess})
	r.Register(&ExecuteCommandTool{allowedCommands: cfg.AllowedCommands})
	// Add other tools like ReadRepo here...

	// Initialize MCP clients and register their tools
	for _, mcpServer := range cfg.AdditionalMCPServers {
		client, err := mcp.NewMCPClient(mcpServer.Name, mcpServer.Command, mcpServer.Args)
		if err != nil {
			// In a real application, you might want to handle this more gracefully
			// than just printing, but for now, this is fine.
			fmt.Printf("ERROR: Failed to initialize MCP client for '%s': %v\n", mcpServer.Name, err)
			continue
		}
		r.mcpClients[mcpServer.Name] = client
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
		if strings.Contains(toolName, ".") {
			parts := strings.SplitN(toolName, ".", 2)
			if len(parts) != 2 {
				return nil, errors.New("invalid MCP tool format '%s' in toolset '%s'", toolName, ts.Name)
			}
			serverName, mcpToolName := parts[0], parts[1]

			client, ok := r.mcpClients[serverName]
			if !ok {
				return nil, errors.New("MCP server '%s' for tool '%s' not registered", serverName, toolName)
			}
			if t, ok := client.GetTool(mcpToolName); ok {
				activeTools = append(activeTools, t)
			} else {
				return nil, errors.New("MCP tool '%s' not found on server '%s'", mcpToolName, serverName)
			}
			continue
		}

		if t, ok := r.GetTool(toolName); ok {
			activeTools = append(activeTools, t)
		} else {
			return nil, errors.New("tool '%s' from toolset '%s' is not registered", toolName, ts.Name)
		}
	}
	return activeTools, nil
}

// isPathRestricted checks if a path matches any of the glob patterns.
func isPathRestricted(path string, patterns []string) (bool, error) {
	for _, pattern := range patterns {
		match, err := doublestar.PathMatch(pattern, path)
		if err != nil {
			return false, errors.Wrapf(err, "invalid glob pattern '%s'", pattern)
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
