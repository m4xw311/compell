package terminal

import (
	"context"
	"testing"

	"github.com/m4xw311/compell/agent"
	"github.com/m4xw311/compell/config"
	"github.com/m4xw311/compell/llm"
	"github.com/m4xw311/compell/session"
)

// createTestConfig creates a config with a default toolset for testing
func createTestConfig() *config.Config {
	return &config.Config{
		Toolsets: []config.Toolset{
			{
				Name:  "default",
				Tools: []string{},
			},
		},
	}
}

func TestTerminalNew(t *testing.T) {
	// Create a mock agent for testing
	cfg := createTestConfig()
	sess, err := session.New("test-session")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	mockClient := &llm.MockLLMClient{}
	testAgent, err := agent.New(cfg, sess, "default", agent.ModeAuto, mockClient, agent.ToolVerbosityNone)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	// Create terminal instance
	term := New(testAgent)
	if term == nil {
		t.Fatal("Expected terminal instance, got nil")
	}

	if term.agent != testAgent {
		t.Fatal("Terminal agent doesn't match the provided agent")
	}
}

func TestTerminalProcessTurn(t *testing.T) {
	// Create a mock agent for testing
	cfg := createTestConfig()
	sess, err := session.New("test-session")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	mockClient := &llm.MockLLMClient{}
	testAgent, err := agent.New(cfg, sess, "default", agent.ModeAuto, mockClient, agent.ToolVerbosityNone)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	term := New(testAgent)

	// Test processTurn with a simple input
	// Note: This is a basic test that verifies the method doesn't panic
	// More comprehensive testing would require mocking stdin/stdout
	ctx := context.Background()

	// Since processTurn calls ProcessUserInput which will use the LLM client,
	// we expect it to process without errors with the mock client
	err = term.processTurn(ctx, "test input")
	if err != nil {
		t.Errorf("processTurn failed: %v", err)
	}
}

func TestTerminalCallbacks(t *testing.T) {
	// This test verifies that the terminal creates appropriate callbacks
	// when processing user input

	cfg := createTestConfig()
	mockClient := &llm.MockLLMClient{}

	// Test with different verbosity levels
	testCases := []struct {
		name      string
		mode      agent.Mode
		verbosity agent.ToolVerbosity
	}{
		{"AutoModeNoVerbosity", agent.ModeAuto, agent.ToolVerbosityNone},
		{"AutoModeInfoVerbosity", agent.ModeAuto, agent.ToolVerbosityInfo},
		{"AutoModeAllVerbosity", agent.ModeAuto, agent.ToolVerbosityAll},
		{"PromptModeNoVerbosity", agent.ModePrompt, agent.ToolVerbosityNone},
		{"PromptModeAllVerbosity", agent.ModePrompt, agent.ToolVerbosityAll},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a new session for each test case to avoid state interference
			testSess, err := session.New("test-session-" + tc.name)
			if err != nil {
				t.Fatalf("Failed to create session: %v", err)
			}

			testAgent, err := agent.New(cfg, testSess, "default", tc.mode, mockClient, tc.verbosity)
			if err != nil {
				t.Fatalf("Failed to create agent: %v", err)
			}

			term := New(testAgent)
			ctx := context.Background()

			// Process a turn - this should create and use callbacks internally
			err = term.processTurn(ctx, "test input for "+tc.name)
			if err != nil {
				t.Errorf("processTurn failed for %s: %v", tc.name, err)
			}
		})
	}
}

func TestTerminalRun(t *testing.T) {
	// Test that Run method properly handles initial prompts
	// Note: Full testing of the interactive loop would require mocking stdin

	cfg := createTestConfig()
	sess, err := session.New("test-session-run")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	mockClient := &llm.MockLLMClient{}
	testAgent, err := agent.New(cfg, sess, "default", agent.ModeAuto, mockClient, agent.ToolVerbosityNone)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	term := New(testAgent)
	ctx := context.Background()

	// Test with initial prompt - should process once and then exit due to no stdin
	// The Run method will exit immediately after processing the initial prompt
	// when stdin is not available (as in test environment)
	err = term.Run(ctx, "initial test prompt")
	if err != nil {
		t.Errorf("Run failed with initial prompt: %v", err)
	}

	// Test without initial prompt - should exit immediately due to no stdin
	err = term.Run(ctx, "")
	if err != nil {
		t.Errorf("Run failed without initial prompt: %v", err)
	}
}
