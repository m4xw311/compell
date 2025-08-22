package mcp

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/m4xw311/compell/errors"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// MCPClient manages the connection to a single MCP server subprocess.
type MCPClient struct {
	Name  string
	cmd   *exec.Cmd
	conn  *mcpsdk.ClientSession
	tools map[string]*MCPTool // Map of tool name (e.g., "file_reader") to the tool instance.
}

// NewMCPClient starts the MCP server subprocess and initializes the client.
// It is responsible for discovering the tools provided by the server.
func NewMCPClient(name, command string, args []string) (*MCPClient, error) {
	cmd := exec.Command(command, args...)
	cmd.Stderr = os.Stderr
	mcpClient := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "mcp-client", Version: "v1.0.0"}, nil)
	ctx := context.Background()
	conn, err := mcpClient.Connect(ctx, mcpsdk.NewCommandTransport(cmd))
	if err != nil {
		cmd.Process.Kill()
		return nil, errors.Wrapf(err, "failed to connect to MCP server '%s'", name)
	}
	client := &MCPClient{
		Name:  name,
		cmd:   cmd,
		conn:  conn,
		tools: make(map[string]*MCPTool),
	}
	toolListParams := &mcpsdk.ListToolsParams{}
	for {
		toolList, err := conn.ListTools(ctx, toolListParams)
		if err != nil {
			// Attempt to stop the process we just started.
			cmd.Process.Kill()
			return nil, errors.Wrapf(err, "failed to list tools from MCP server '%s'", name)
		}

		for _, t := range toolList.Tools {
			client.tools[t.Name] = &MCPTool{
				serverName:  name,
				toolName:    t.Name,
				description: t.Description,
				client:      client,
			}
		}

		if toolList.NextCursor == "" {
			break
		}
		toolListParams.Cursor = toolList.NextCursor
	}

	fmt.Printf("INFO: Initialized MCP client for '%s' with %d tools.\n", name, len(client.tools))
	return client, nil
}

// GetTool returns a specific tool provided by this MCP server by its short name.
func (c *MCPClient) GetTool(toolName string) (*MCPTool, bool) {
	tool, ok := c.tools[toolName]
	return tool, ok
}

// Stop terminates the MCP server subprocess.
func (c *MCPClient) Stop() error {
	if c.conn != nil {
		c.conn.Close()
	}
	if c.cmd != nil && c.cmd.Process != nil {
		fmt.Printf("INFO: Terminating MCP server '%s'\n", c.Name)
		return c.cmd.Process.Kill()
	}
	return nil
}

// MCPTool represents a tool available from an external MCP server.
// It is designed to satisfy the `tools.Tool` interface from the parent package.
type MCPTool struct {
	serverName  string
	toolName    string
	description string
	client      *MCPClient // Reference back to the client managing the connection.
}

// Name returns the fully qualified name of the tool in the format "<server>:<tool>".
func (t *MCPTool) Name() string {
	// Using %s:%s was causing 400 error from Gemini so using %s.%s
	//return fmt.Sprintf("%s.%s", t.serverName, t.toolName)
	return t.toolName
}

// Description returns the tool's description, provided by the MCP server.
func (t *MCPTool) Description() string {
	return t.description
}

// Execute sends the command and arguments to the MCP server and returns the result.
func (t *MCPTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	result, err := t.client.conn.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      t.toolName,
		Arguments: args,
	})
	if err != nil {
		return "", errors.Wrapf(err, "failed to call tool '%s'", t.Name())
	}
	op := ""
	for _, c := range result.Content {
		op += c.(*mcp.TextContent).Text
	}
	return op, nil
}
