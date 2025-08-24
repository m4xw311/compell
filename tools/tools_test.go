package tools

import (
	"testing"

	"github.com/m4xw311/compell/config"
	"github.com/m4xw311/compell/tools/mcp"
)

func TestTools(t *testing.T) {
	// TODO: Add actual test logic here.
	// t.Log("Tools test not yet implemented.")
}

// TestWildcardMCPToolSupport tests the wildcard functionality for MCP tools
func TestWildcardMCPToolSupport(t *testing.T) {
	// Note: This is a basic test to verify that the wildcard functionality
	// is implemented correctly in GetActiveTools. A full integration test
	// would require actual running MCP servers.

	registry := &ToolRegistry{
		tools:      make(map[string]Tool),
		mcpClients: make(map[string]*mcp.MCPClient),
	}

	// Create a mock toolset with wildcard
	ts := &config.Toolset{
		Name:  "test",
		Tools: []string{"gopls.*"}, // This should match all gopls tools
	}

	// Verify that the wildcard pattern is handled correctly
	// (This test would need actual MCP clients to be fully functional)
	_ = registry
	_ = ts
	t.Log("Wildcard MCP tool support test placeholder")
}
