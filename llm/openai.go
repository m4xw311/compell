package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/m4xw311/compell/errors"
	"github.com/m4xw311/compell/session"
	"github.com/m4xw311/compell/tools"
	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
)

// OpenAILLMClient is a client for the OpenAI Chat Completion API.
type OpenAILLMClient struct {
	client *openai.Client
	model  string
}

// NewOpenAILLMClient creates a new OpenAILLMClient. It requires the OPENAI_API_KEY environment variable to be set.
// It also supports OPENAI_BASE_URL for custom API endpoints.
func NewOpenAILLMClient(ctx context.Context, modelName string) (*OpenAILLMClient, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, errors.New("OPENAI_API_KEY environment variable not set")
	}

	// Create client options
	options := []option.RequestOption{
		option.WithAPIKey(apiKey),
	}

	// Check for custom base URL
	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL != "" {
		options = append(options, option.WithBaseURL(baseURL))
	}

	// The v2 SDK uses functional options for configuration.
	c := openai.NewClient(options...)
	// The &c is required, dn not replace and just use c
	return &OpenAILLMClient{client: &c, model: modelName}, nil
}

// Chat sends a chat request to OpenAI and converts the response into our internal session.Message format.
func (o *OpenAILLMClient) Chat(ctx context.Context, messages []session.Message, availableTools []tools.Tool) (*session.Message, error) {
	// Convert internal messages to OpenAI chat messages.
	chatMessages := convertMessagesToOpenaiContent(messages)

	params := openai.ChatCompletionNewParams{
		Model:    openai.ChatModel(o.model),
		Messages: chatMessages,
		Tools:    convertToolsToOpenAITools(availableTools),
	}

	resp, err := o.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to send message to OpenAI")
	}

	return processOpenaiResponse(resp)
}

// processOpenaiResponse converts an OpenAI API response into our internal session.Message format.
func processOpenaiResponse(resp *openai.ChatCompletion) (*session.Message, error) {
	if len(resp.Choices) == 0 {
		return &session.Message{Role: "assistant", Content: ""}, nil
	}

	choice := resp.Choices[0].Message

	// If model requests tool calls, the ToolCalls field will be present.
	if len(choice.ToolCalls) > 0 {
		var sessToolCalls []session.ToolCall
		for _, tc := range choice.ToolCalls {
			var toolArgs map[string]interface{}
			// Arguments are a JSON string; we expect it to be a flat map of arguments.
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &toolArgs); err != nil {
				return nil, errors.Wrapf(err, "failed to unmarshal function call arguments from OpenAI")
			}
			sessToolCalls = append(sessToolCalls, session.ToolCall{
				ToolCallID: tc.ID,
				Name:       tc.Function.Name,
				Args:       toolArgs,
			})
		}
		return &session.Message{
			Role:      "assistant",
			Content:   choice.Content,
			ToolCalls: sessToolCalls,
		}, nil
	}

	// Otherwise, return a normal assistant text response.
	return &session.Message{Role: "assistant", Content: choice.Content}, nil
}

// convertMessagesToOpenaiContent converts our internal message format to OpenAI's.
func convertMessagesToOpenaiContent(messages []session.Message) []openai.ChatCompletionMessageParamUnion {
	var chatMessages []openai.ChatCompletionMessageParamUnion
	for _, msg := range messages {
		switch msg.Role {
		case "assistant":
			assistantMessage := openai.ChatCompletionMessage{
				Role:    "assistant",
				Content: msg.Content,
			}
			if len(msg.ToolCalls) > 0 {
				var toolCalls []openai.ChatCompletionMessageToolCallUnion
				for _, tc := range msg.ToolCalls {
					argsBytes, err := json.Marshal(tc.Args)
					if err != nil {
						fmt.Printf("Warning: could not marshal tool call arguments for %s: %v. Skipping function call in history.\n", tc.Name, err)
						continue
					}
					toolCalls = append(toolCalls, openai.ChatCompletionMessageToolCallUnion{
						ID:   tc.ToolCallID,
						Type: "function",
						Function: openai.ChatCompletionMessageFunctionToolCallFunction{
							Name:      tc.Name,
							Arguments: string(argsBytes),
						},
					})
				}
				assistantMessage.ToolCalls = toolCalls
			}
			chatMessages = append(chatMessages, assistantMessage.ToParam())
		case "tool":
			// A "tool" role message corresponds to a "tool" role message in the OpenAI API.
			if len(msg.ToolCalls) != 1 {
				fmt.Printf("Warning: tool message is malformed; expected exactly one ToolCall to identify the function name, but found %d. Skipping.\n", len(msg.ToolCalls))
				continue
			}
			chatMessages = append(chatMessages, openai.ToolMessage(msg.Content, msg.ToolCalls[0].ToolCallID))
		case "user":
			fallthrough
		default:
			chatMessages = append(chatMessages, openai.UserMessage(msg.Content))
		}
	}
	return chatMessages
}

// convertToolsToOpenAITools converts our Tool interface to the OpenAI Tool format.
func convertToolsToOpenAITools(ts []tools.Tool) []openai.ChatCompletionToolUnionParam {
	if len(ts) == 0 {
		return nil
	}
	var openAITools []openai.ChatCompletionToolUnionParam
	for _, t := range ts {
		// Unlike Gemini, OpenAI models work better when the parameters are not nested.
		// We define a generic object schema and let the model infer the arguments.
		params := openai.FunctionParameters{
			"type":       "object",
			"properties": map[string]any{},
		}

		toolParam := openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
			Name:        t.Name(),
			Description: openai.String(t.Description()),
			Parameters:  params,
		})
		openAITools = append(openAITools, toolParam)
	}
	return openAITools
}
