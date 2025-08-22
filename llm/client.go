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

// MockLLMClient is a placeholder for testing.
type MockLLMClient struct{}

func (m *MockLLMClient) Chat(ctx context.Context, messages []session.Message, availableTools []tools.Tool) (*session.Message, error) {
	fmt.Println("\n--- MOCK LLM CLIENT ---")
	fmt.Printf("Received %d messages. Last message: '%s'\n", len(messages), messages[len(messages)-1].Content)
	var toolNames []string
	for _, tool := range availableTools {
		toolNames = append(toolNames, tool.Name())
	}
	fmt.Printf("Available tools: %v\n", toolNames)
	fmt.Println("Returning a mock response.")
	fmt.Println("-----------------------")

	// This mock will just parrot back the user's last message.
	// A real implementation would make an API call here.
	lastUserMessage := messages[len(messages)-1].Content
	return &session.Message{
		Role:    "assistant",
		Content: fmt.Sprintf("I am a mock LLM. You said: '%s'. I cannot use tools yet.", lastUserMessage),
	}, nil
}
