package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/m4xw311/compell/errors"
	"github.com/m4xw311/compell/session"
	"github.com/m4xw311/compell/tools"
)

// AnthropicLLMClient is a client for the Anthropic API.
type AnthropicLLMClient struct {
	client *anthropic.Client
	model  string
}

// NewAnthropicLLMClient creates a new AnthropicLLMClient.
// It requires the ANTHROPIC_API_KEY environment variable to be set.
func NewAnthropicLLMClient(ctx context.Context, modelName string) (*AnthropicLLMClient, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return nil, errors.New("ANTHROPIC_API_KEY environment variable not set")
	}

	client := anthropic.NewClient(
		option.WithAPIKey(apiKey),
	)

	return &AnthropicLLMClient{
		client: &client,
		model:  modelName,
	}, nil
}

// Chat sends a chat request to the Anthropic API.
func (a *AnthropicLLMClient) Chat(ctx context.Context, messages []session.Message, availableTools []tools.Tool) (*session.Message, error) {
	// Convert session messages to Anthropic format
	anthropicMessages, systemPrompt := convertMessagesToAnthropicMessages(messages)

	// Convert available tools to Anthropic format
	anthropicTools := convertToolsToAnthropicTools(availableTools)

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(a.model),
		MaxTokens: 4096,
		Messages:  anthropicMessages,
	}

	if systemPrompt != "" {
		params.System = []anthropic.TextBlockParam{
			{Text: systemPrompt},
		}
	}
	params.Tools = make([]anthropic.ToolUnionParam, len(anthropicTools))
	for i, toolParam := range anthropicTools {
		params.Tools[i] = anthropic.ToolUnionParam{OfTool: &toolParam}
	}

	resp, err := a.client.Messages.New(ctx, params)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to send message to Anthropic")
	}

	// Process the response from Anthropic
	return processAnthropicResponse(resp)
}

// convertMessagesToAnthropicMessages converts our internal message format to Anthropic's format.
func convertMessagesToAnthropicMessages(messages []session.Message) ([]anthropic.MessageParam, string) {
	var anthropicMessages []anthropic.MessageParam
	var systemPrompt string

	for _, msg := range messages {
		switch msg.Role {
		case "user":
			anthropicMessages = append(anthropicMessages, anthropic.NewUserMessage(
				anthropic.NewTextBlock(msg.Content),
			))
		case "assistant":
			if len(msg.ToolCalls) > 0 {
				// Handle tool calls
				var contentItems []anthropic.ContentBlockParamUnion
				for _, tc := range msg.ToolCalls {
					argsBytes, err := json.Marshal(tc.Args)
					if err != nil {
						fmt.Printf("Warning: could not marshal tool call arguments for %s: %v. Skipping.\n", tc.Name, err)
						continue
					}

					contentItems = append(contentItems, anthropic.ContentBlockParamUnion{
						OfToolUse: &anthropic.ToolUseBlockParam{
							Type:  "tool_use",
							ID:    tc.ToolCallID,
							Name:  tc.Name,
							Input: argsBytes,
						}})
				}

				anthropicMessages = append(anthropicMessages, anthropic.MessageParam{
					Role:    anthropic.MessageParamRoleAssistant,
					Content: contentItems,
				})
			} else if msg.Content != "" {
				// Handle regular assistant messages
				anthropicMessages = append(anthropicMessages, anthropic.MessageParam{
					Role: anthropic.MessageParamRoleAssistant,
					Content: []anthropic.ContentBlockParamUnion{{
						OfText: &anthropic.TextBlockParam{
							Text: msg.Content,
						},
					}},
				})
			}
		case "tool":
			// Handle tool responses
			if len(msg.ToolCalls) > 0 {
				anthropicMessages = append(anthropicMessages, anthropic.MessageParam{
					Role: anthropic.MessageParamRoleUser,
					Content: []anthropic.ContentBlockParamUnion{{
						OfToolResult: &anthropic.ToolResultBlockParam{
							ToolUseID: msg.ToolCalls[0].ToolCallID,
							Content: []anthropic.ToolResultBlockParamContentUnion{{
								OfText: &anthropic.TextBlockParam{
									Text: msg.Content,
								},
							}},
						},
					},
					}})
			}
		case "system":
			// Handle system messages (take the last one as the system prompt)
			systemPrompt = msg.Content
		}
	}

	return anthropicMessages, systemPrompt
}

// convertToolsToAnthropicTools converts our Tool interface to Anthropic's tool format.
func convertToolsToAnthropicTools(ts []tools.Tool) []anthropic.ToolParam {
	if len(ts) == 0 {
		return nil
	}

	var anthropicTools []anthropic.ToolParam
	for _, t := range ts {
		anthropicTools = append(anthropicTools, anthropic.ToolParam{
			Name:        t.Name(),
			Description: anthropic.String(t.Description()),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{},
			},
		})
	}
	return anthropicTools
}

// processAnthropicResponse converts an Anthropic API response into our internal session.Message format.
func processAnthropicResponse(resp *anthropic.Message) (*session.Message, error) {
	if len(resp.Content) == 0 {
		return &session.Message{Role: "assistant", Content: ""}, nil
	}

	var responseContent string
	var toolCalls []session.ToolCall

	for _, content := range resp.Content {
		switch c := content.AsAny().(type) {
		case anthropic.TextBlock:
			responseContent += c.Text
		case anthropic.ToolUseBlock:
			// Extract tool call information
			var args map[string]interface{}
			if err := json.Unmarshal(c.Input, &args); err != nil {
				return nil, errors.Wrapf(err, "failed to unmarshal tool call input")
			}

			toolCall := session.ToolCall{
				ToolCallID: c.ID,
				Name:       c.Name,
				Args:       args,
			}
			toolCalls = append(toolCalls, toolCall)
		}
	}

	return &session.Message{
		Role:      "assistant",
		Content:   responseContent,
		ToolCalls: toolCalls,
	}, nil
}
