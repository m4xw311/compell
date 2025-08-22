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
		role := "user" // Default to user
		if msg.Role == "assistant" {
			role = "model"
		}
		// Note: The "tool" role needs special handling if we were to process tool responses,
		// which would typically be appended as a genai.Part in a new user message.
		contents = append(contents, &genai.Content{
			Role:  role,
			Parts: []genai.Part{genai.Text(msg.Content)},
		})
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
		return nil, errors.New("received an empty response from Gemini")
	}

	content := resp.Candidates[0].Content
	var responseContent string

	for _, part := range content.Parts {
		switch v := part.(type) {
		case genai.Text:
			responseContent += string(v)
		case genai.FunctionCall:
			// Find the tool that the model wants to call.
			var calledTool tools.Tool
			for _, tool := range availableTools {
				if tool.Name() == v.Name {
					calledTool = tool
					break
				}
			}

			// If the tool is not found, report an error back to the model. This should
			// not happen if the model is behaving correctly.
			if calledTool == nil {
				responseContent += fmt.Sprintf("Error: model requested to call unavailable tool '%s'", v.Name)
				continue
			}

			// Extract the arguments. As defined in `convertToolsToGeminiTools`,
			// the arguments are nested under an "args" key.
			toolArgs, ok := v.Args["args"].(map[string]interface{})
			if !ok {
				responseContent += fmt.Sprintf("Error: invalid arguments for tool '%s', expected a map under 'args' key", v.Name)
				continue
			}

			// Execute the tool.
			result, err := calledTool.Execute(ctx, toolArgs)
			if err != nil {
				// Report tool execution error back to the model.
				responseContent += fmt.Sprintf("Error executing tool '%s': %v", v.Name, err)
				continue
			}

			// Append the tool's result to the response content.
			// A more complete implementation might add a new message with role "tool"
			// to the session history, but for now this lets the model see the result.
			responseContent += result
		default:
			return nil, errors.New("unsupported part type in Gemini response: %T", v)
		}
	}

	return &session.Message{
		Role:    "assistant",
		Content: responseContent,
	}, nil
}
