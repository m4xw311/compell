package acp

import (
	"bufio"
	"bytes"
	"context"
	"testing"

	"github.com/m4xw311/compell/agent"
	"github.com/m4xw311/compell/config"
	"github.com/m4xw311/compell/llm"
	"github.com/m4xw311/compell/session"
)

// TestACPInit tests the ACP init request/response
func TestACPInit(t *testing.T) {
	// Create a mock configuration
	cfg := &config.Config{
		LLMClient: "mock",
		Model:     "mock-model",
		Toolsets: []config.Toolset{
			config.Toolset{
				Name: "default",
				Tools: []string{
					"read_dir",
					"read_file",
					"write_file",
					"execute_command",
				},
			},
		},
	}

	// Create a mock session
	sess, err := session.New("test-acp-session")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Create a mock LLM client
	client := &llm.MockLLMClient{}

	// Create the agent
	compellAgent, err := agent.New(cfg, sess, "default", agent.ModePrompt, client, agent.ToolVerbosityNone)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Test the runACPMode function
	ctx := context.Background()
	noTrace := false

	var mockStdinBuff bytes.Buffer
	var mockStdoutBuff bytes.Buffer
	in := bufio.NewReader(&mockStdinBuff)
	out := bufio.NewWriter(&mockStdoutBuff)

	err = Run(ctx, compellAgent, in, out, &noTrace)
	if err != nil {
		t.Fatalf("Failed to run ACP mode: %v", err)
	}

	// Simulate the initialization message from the client
	mockStdinBuff.WriteString(`{"id":0,"method":"initialize","params":{"protocolVersion":1,"clientCapabilities":{"fs":{"readTextFile":true,"writeTextFile":true}}}}`)
	// Read all contents of mock strout into a byte slice.
	got := mockStdinBuff.Bytes()
	// ToDo; String compare like this feels brittle
	//  Parse json and compare all fields
	expected := `{"id":0,"method":"initialize","params":{"protocolVersion":1,"clientCapabilities":{"fs":{"readTextFile":true,"writeTextFile":true}}}}`
	if string(got) != expected {
		t.Errorf("Unexpected data received: %s", string(got))
	}

}
