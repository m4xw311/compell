package llm

import (
	"context"
	"testing"

	"github.com/m4xw311/compell/session"
	"github.com/m4xw311/compell/tools"
)

// MockTool is a simple mock tool for testing
type MockTool struct {
	name        string
	description string
}

func (m *MockTool) Name() string {
	return m.name
}

func (m *MockTool) Description() string {
	return m.description
}

func (m *MockTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	return "mock result", nil
}

func TestConvertMessagesToAnthropicFormat(t *testing.T) {
	// Test user message
	messages := []session.Message{
		{
			Role:    "user",
			Content: "Hello, world!",
		},
	}

	result, _ := convertMessagesToAnthropicFormat(messages)
	if len(result) != 1 {
		t.Errorf("Expected 1 message, got %d", len(result))
	}

	if result[0]["role"] != "user" {
		t.Errorf("Expected role 'user', got '%s'", result[0]["role"])
	}

	// Test assistant message with content
	messages = []session.Message{
		{
			Role:    "assistant",
			Content: "Hello! How can I help you?",
		},
	}

	result, _ = convertMessagesToAnthropicFormat(messages)
	if len(result) != 1 {
		t.Errorf("Expected 1 message, got %d", len(result))
	}

	if result[0]["role"] != "assistant" {
		t.Errorf("Expected role 'assistant', got '%s'", result[0]["role"])
	}

	// Test assistant message with tool calls
	messages = []session.Message{
		{
			Role: "assistant",
			ToolCalls: []session.ToolCall{
				{
					ToolCallID: "call_1",
					Name:       "test_tool",
					Args: map[string]interface{}{
						"param1": "value1",
					},
				},
			},
		},
	}

	result, _ = convertMessagesToAnthropicFormat(messages)
	if len(result) != 1 {
		t.Errorf("Expected 1 message, got %d", len(result))
	}

	// Test tool response message
	messages = []session.Message{
		{
			Role:    "tool",
			Content: "Tool result",
			ToolCalls: []session.ToolCall{
				{
					ToolCallID: "call_1",
					Name:       "test_tool",
				},
			},
		},
	}

	result, _ = convertMessagesToAnthropicFormat(messages)
	if len(result) != 1 {
		t.Errorf("Expected 1 message, got %d", len(result))
	}

	if result[0]["role"] != "user" {
		t.Errorf("Expected role 'user', got '%s'", result[0]["role"])
	}
}

func TestCreateAnthropicRequest(t *testing.T) {
	messages := []map[string]interface{}{
		{
			"role": "user",
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": "Hello!",
				},
			},
		},
	}

	// Test with no tools
	body, err := createAnthropicRequest(messages, "", nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(body) == 0 {
		t.Error("Expected non-empty request body")
	}

	// Test with tools
	tools := []tools.Tool{
		&MockTool{
			name:        "test_tool",
			description: "A test tool",
		},
	}

	body, err = createAnthropicRequest(messages, "", tools)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(body) == 0 {
		t.Error("Expected non-empty request body")
	}
}
