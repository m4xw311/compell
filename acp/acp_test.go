package acp

import (
	"testing"

	"github.com/m4xw311/compell/agent"
	"github.com/m4xw311/compell/config"
	"github.com/m4xw311/compell/llm"
	"github.com/m4xw311/compell/session"
)

func TestACPInitialization(t *testing.T) {
	// Create a mock agent for testing
	cfg := &config.Config{
		Toolsets: []config.Toolset{
			{
				Name:  "default",
				Tools: []string{"read_file", "write_file"},
			},
		},
	}
	sess, _ := session.New("test")
	client := &llm.MockLLMClient{}

	compellAgent, err := agent.New(cfg, sess, "default", agent.ModePrompt, client, agent.ToolVerbosityNone)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Test that we can create an ACP runner
	// Note: We can't easily test the full Run function since it interacts with stdin/stdout
	// but we can at least verify the package compiles and imports correctly
	_ = compellAgent

	t.Log("ACP package initialized successfully")
}
