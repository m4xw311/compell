package llm

import (
	"context"
	"fmt"
	"os"

	"github.com/google/generative-ai-go/genai"
	"github.com/m4xw311/compell/errors"
	"github.com/m4xw311/compell/session"
	"github.com/m4xw311/compell/tools"
	"google.golang.org/api/option"
)

// GeminiLLMClient is a client for the Google Gemini API.
type GeminiLLMClient struct {
	model *genai.GenerativeModel
}

// NewGeminiLLMClient creates a new GeminiLLMClient.
// It requires the GEMINI_API_KEY environment variable to be set.
func NewGeminiLLMClient(ctx context.Context, modelName string) (*GeminiLLMClient, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, errors.New("GEMINI_API_KEY environment variable not set")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create genai client")
	}

	model := client.GenerativeModel(modelName)

	return &GeminiLLMClient{
		model: model,
	}, nil
}

// Chat sends a chat rexquest to the Gemini API.
func (g *GeminiLLMClient) Chat(ctx context.Context, messages []session.Message, availableTools []tools.Tool) (*session.Message, error) {
	// Convert session messages to Gemini's content format.
	history := convertMessagesToGeminiContent(messages)

	// Convert available tools to Gemini's tool format.
	geminiTools := convertToolsToGeminiTools(availableTools)
	g.model.Tools = geminiTools

	// The last message is the new prompt.
	lastMessage := history[len(history)-1]

	chatSession := g.model.StartChat()
	chatSession.History = history[:len(history)-1]
	resp, err := chatSession.SendMessage(ctx, lastMessage.Parts...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to send message to Gemini")
	}

	// Process the response from Gemini.
	return processGeminiResponse(ctx, resp, availableTools)
}

// convertMessagesToGeminiContent converts our internal message format to Gemini's.
func convertMessagesToGeminiContent(messages []session.Message) []*genai.Content {
	var contents []*genai.Content
	for _, msg := range messages {
		role := "user" // Default role
		var parts []genai.Part

		switch msg.Role {
		case "assistant":
			role = "model"
			if msg.Content != "" {
				parts = append(parts, genai.Text(msg.Content))
			}
			for _, tc := range msg.ToolCalls {
				parts = append(parts, genai.FunctionCall{
					Name: tc.Name,
					// The arguments from the model are nested under an "args" key,
					// so we replicate that structure when adding to history.
					Args: map[string]interface{}{"args": tc.Args},
				})
			}
		case "tool":
			role = "user" // Tool responses are sent with the 'user' role to Gemini.
			// This assumes the agent creates a "tool" message where Content is the
			// result and ToolCalls[0].Name is the name of the tool that was called.
			if len(msg.ToolCalls) != 1 {
				fmt.Printf("Warning: tool message is malformed; expected exactly one ToolCall to identify the function name, but found %d. Skipping.\n", len(msg.ToolCalls))
				continue // Skip this malformed message
			}
			toolName := msg.ToolCalls[0].Name
			parts = append(parts, genai.FunctionResponse{
				Name: toolName,
				// The response needs to be a JSON-serializable map or struct.
				// We wrap the raw string output in a map.
				Response: map[string]interface{}{"output": msg.Content},
			})
		case "user":
			fallthrough
		default:
			role = "user"
			if msg.Content != "" {
				parts = append(parts, genai.Text(msg.Content))
			}
		}

		if len(parts) > 0 {
			contents = append(contents, &genai.Content{
				Role:  role,
				Parts: parts,
			})
		}
	}
	return contents
}

// convertToolsToGeminiTools converts our Tool interface to Gemini's FunctionDeclaration format.
func convertToolsToGeminiTools(ts []tools.Tool) []*genai.Tool {
	if len(ts) == 0 {
		return nil
	}
	var geminiTools []*genai.Tool
	var funcDecls []*genai.FunctionDeclaration

	for _, tool := range ts {
		// For now, we assume every tool takes a generic map of string-to-any arguments.
		// A more advanced implementation might involve extending the Tool interface
		// to provide a more detailed JSON schema for its parameters.
		fd := &genai.FunctionDeclaration{
			Name:        tool.Name(),
			Description: tool.Description(),
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					"args": {
						Type:        genai.TypeObject,
						Description: "Arguments for the function call, as a map.",
					},
				},
				Required: []string{"args"},
			},
		}
		funcDecls = append(funcDecls, fd)
	}
	geminiTools = append(geminiTools, &genai.Tool{FunctionDeclarations: funcDecls})
	return geminiTools
}

// processGeminiResponse converts a Gemini API response into our internal session.Message format.
func processGeminiResponse(ctx context.Context, resp *genai.GenerateContentResponse, availableTools []tools.Tool) (*session.Message, error) {
	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		// It's possible the model just returned a finish reason like "STOP"
		// with no content. We can check FinishReason and handle if needed.
		// For now, returning an empty message is safe, the agent loop will handle it.
		return &session.Message{Role: "assistant", Content: ""}, nil
	}

	content := resp.Candidates[0].Content
	var responseContent string
	var toolCalls []session.ToolCall
	toolCallIDCounter := 0

	for _, part := range content.Parts {
		switch v := part.(type) {
		case genai.Text:
			responseContent += string(v)
		case genai.FunctionCall:
			// The model has requested to call a tool.
			// We package this into our internal ToolCall struct and pass it to the agent.
			toolArgs, ok := v.Args["args"].(map[string]interface{})
			if !ok {
				// This indicates a malformed request from the LLM based on our tool definition.
				// For now, we'll log this and continue, but a more robust
				// error handling strategy might be to return an error to the agent.
				fmt.Printf("Warning: invalid arguments for tool '%s', expected a map under 'args' key\n", v.Name)
				continue
			}

			// We need to generate a unique ID for each tool call to track the response.
			toolCall := session.ToolCall{
				ToolCallID: fmt.Sprintf("call_%d_%s", toolCallIDCounter, v.Name),
				Name:       v.Name,
				Args:       toolArgs,
			}
			toolCalls = append(toolCalls, toolCall)
			toolCallIDCounter++
		default:
			return nil, errors.New("unsupported part type in Gemini response: %T", v)
		}
	}

	return &session.Message{
		Role:      "assistant",
		Content:   responseContent,
		ToolCalls: toolCalls,
	}, nil
}
