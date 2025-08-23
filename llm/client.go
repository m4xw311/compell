package llm

import (
	"context"
	"fmt"

	"github.com/m4xw311/compell/session"
	"github.com/m4xw311/compell/tools"
)

// LLMClient is the interface for interacting with a Large Language Model.
type LLMClient interface {
	Chat(ctx context.Context, messages []session.Message, availableTools []tools.Tool) (*session.Message, error)
}

// MockLLMClient is a placeholder for testing that can be configured to
// return specific responses, including text and tool calls.
type MockLLMClient struct {
	MockResponseContent string
	MockToolCalls       []session.ToolCall
	MockToolResponse    string // The response to return after a tool call is processed.
	ReturnToolCall      bool
	ToolNameToCall      string
	ToolArgsToCall      map[string]interface{}
}

func (m *MockLLMClient) Chat(ctx context.Context, messages []session.Message, availableTools []tools.Tool) (*session.Message, error) {
	fmt.Println("\n--- MOCK LLM CLIENT ---")
	fmt.Printf("Received %d messages.\n", len(messages))

	// Check if the last message is a tool response
	if len(messages) > 0 && messages[len(messages)-1].Role == "tool" {
		fmt.Println("MockLLMClient: Received tool response. Returning configured MockToolResponse.")
		return &session.Message{
			Role:    "assistant",
			Content: m.MockToolResponse,
		}, nil
	}

	if m.ReturnToolCall {
		fmt.Println("MockLLMClient: Returning a mock tool call.")
		toolCall := session.ToolCall{
			ToolCallID: "mock_call_1",
			Name:       m.ToolNameToCall,
			Args:       m.ToolArgsToCall,
		}
		return &session.Message{
			Role:      "assistant",
			ToolCalls: []session.ToolCall{toolCall},
		}, nil
	}

	fmt.Println("MockLLMClient: Returning a mock text response.")
	return &session.Message{
		Role:    "assistant",
		Content: m.MockResponseContent,
	}, nil
}
