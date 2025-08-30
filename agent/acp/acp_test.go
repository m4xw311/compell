package acp

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
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

// TestExtractUserTextWithResourceLink tests the extractUserText function with ResourceLink content blocks
func TestExtractUserTextWithResourceLink(t *testing.T) {
	// Create test data directory and file
	testDir := "./testdata"
	testFile := testDir + "/test.txt"
	testContent := "This is test file content"

	// Ensure test directory exists
	err := os.MkdirAll(testDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir) // Clean up after test

	// Create test file
	err = os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Convert to file URI
	absPath, _ := filepath.Abs(testFile)
	fileURI := "file://" + absPath

	// Test cases
	tests := []struct {
		name     string
		blocks   []contentBlock
		expected string
		contains []string
	}{
		{
			name: "text only",
			blocks: []contentBlock{
				{Type: "text", Text: "Hello"},
				{Type: "text", Text: "World"},
			},
			expected: "Hello\nWorld",
		},
		{
			name: "resource_link with file",
			blocks: []contentBlock{
				{Type: "text", Text: "Check this file:"},
				{
					Type:        "resource_link",
					URI:         fileURI,
					Name:        "test.txt",
					MimeType:    "text/plain",
					Title:       "Test File",
					Description: "A test file",
				},
			},
			contains: []string{
				"Check this file:",
				"=== Resource: test.txt ===",
				"Title: Test File",
				"Description: A test file",
				"URI: file://",
				"Type: text/plain",
				"--- File Contents ---",
				testContent,
				"--- End of File ---",
			},
		},
		{
			name: "resource_link with non-file URI",
			blocks: []contentBlock{
				{
					Type:     "resource_link",
					URI:      "https://example.com/file.txt",
					Name:     "remote.txt",
					MimeType: "text/plain",
				},
			},
			contains: []string{
				"=== Resource: remote.txt ===",
				"URI: https://example.com/file.txt",
				"[External resource - content not available]",
			},
		},
		{
			name: "mixed content",
			blocks: []contentBlock{
				{Type: "text", Text: "Start"},
				{
					Type: "resource_link",
					URI:  "https://example.com/doc.pdf",
					Name: "document.pdf",
				},
				{Type: "text", Text: "End"},
			},
			contains: []string{
				"Start",
				"=== Resource: document.pdf ===",
				"End",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractUserText(tt.blocks)

			if tt.expected != "" {
				if result != tt.expected {
					t.Errorf("extractUserText() = %q, want %q", result, tt.expected)
				}
			}

			for _, substr := range tt.contains {
				if !strings.Contains(result, substr) {
					t.Errorf("extractUserText() result does not contain %q\nGot: %q", substr, result)
				}
			}
		})
	}
}
