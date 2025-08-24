package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/m4xw311/compell/errors"
	"github.com/m4xw311/compell/session"
	"github.com/m4xw311/compell/tools"
)

// BedrockLLMClient is a client for the Anthropic models on AWS Bedrock.
type BedrockLLMClient struct {
	client   *bedrockruntime.Client
	modelID  string
	region   string
	endpoint string
}

// NewBedrockLLMClient creates a new BedrockLLMClient.
// It requires AWS credentials to be configured in the environment.
func NewBedrockLLMClient(ctx context.Context, modelID string) (*BedrockLLMClient, error) {
	// Get AWS configuration
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load AWS config")
	}

	// Create Bedrock Runtime client
	client := bedrockruntime.NewFromConfig(cfg)

	// Get region from config or environment
	region := cfg.Region
	if region == "" {
		region = os.Getenv("AWS_DEFAULT_REGION")
	}
	if region == "" {
		region = os.Getenv("AWS_REGION")
	}
	if region == "" {
		region = "us-east-1" // Default region
	}

	// Get custom endpoint if specified (useful for testing)
	endpoint := os.Getenv("BEDROCK_ENDPOINT_URL")

	return &BedrockLLMClient{
		client:   client,
		modelID:  modelID,
		region:   region,
		endpoint: endpoint,
	}, nil
}

// Chat sends a chat request to the Anthropic model via AWS Bedrock.
func (b *BedrockLLMClient) Chat(ctx context.Context, messages []session.Message, availableTools []tools.Tool) (*session.Message, error) {
	// Convert session messages to Anthropic format
	anthropicMessages, systemPrompt := convertMessagesToAnthropicFormat(messages)

	// Create the request body for Anthropic on Bedrock
	requestBody, err := createAnthropicRequest(anthropicMessages, systemPrompt, availableTools)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create Anthropic request")
	}

	// Send request to Bedrock
	resp, err := b.client.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(b.modelID),
		ContentType: aws.String("application/json"),
		Body:        requestBody,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to invoke Bedrock model")
	}

	// Process the response from Bedrock
	return processBedrockResponse(resp.Body, availableTools)
}

// convertMessagesToAnthropicFormat converts our internal message format to Anthropic's format.
func convertMessagesToAnthropicFormat(messages []session.Message) ([]map[string]interface{}, string) {
	var anthropicMessages []map[string]interface{}
	var systemPrompt string

	for _, msg := range messages {
		switch msg.Role {
		case "user":
			anthropicMessages = append(anthropicMessages, map[string]interface{}{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type": "text",
						"text": msg.Content,
					},
				},
			})
		case "assistant":
			if len(msg.ToolCalls) > 0 {
				// Handle tool calls
				var toolUses []map[string]interface{}
				for _, tc := range msg.ToolCalls {
					toolUses = append(toolUses, map[string]interface{}{
						"type":  "tool_use",
						"id":    tc.ToolCallID,
						"name":  tc.Name,
						"input": tc.Args,
					})
				}

				anthropicMessages = append(anthropicMessages, map[string]interface{}{
					"role":    "assistant",
					"content": toolUses,
				})
			} else if msg.Content != "" {
				// Handle regular assistant messages
				anthropicMessages = append(anthropicMessages, map[string]interface{}{
					"role": "assistant",
					"content": []map[string]interface{}{
						{
							"type": "text",
							"text": msg.Content,
						},
					},
				})
			}
		case "tool":
			// Handle tool responses
			if len(msg.ToolCalls) > 0 {
				anthropicMessages = append(anthropicMessages, map[string]interface{}{
					"role": "user",
					"content": []map[string]interface{}{
						{
							"type":        "tool_result",
							"tool_use_id": msg.ToolCalls[0].ToolCallID,
							"content":     msg.Content,
						},
					},
				})
			}
		}
	}

	return anthropicMessages, systemPrompt
}

// createAnthropicRequest creates the request body for Anthropic models on Bedrock.
func createAnthropicRequest(messages []map[string]interface{}, systemPrompt string, availableTools []tools.Tool) ([]byte, error) {
	request := map[string]interface{}{
		"anthropic_version": "bedrock-2023-05-31",
		"max_tokens":        4096,
		"messages":          messages,
	}

	if systemPrompt != "" {
		request["system"] = systemPrompt
	}

	if len(availableTools) > 0 {
		var tools []map[string]interface{}
		for _, tool := range availableTools {
			tools = append(tools, map[string]interface{}{
				"name":        tool.Name(),
				"description": tool.Description(),
				"input_schema": map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				},
			})
		}
		request["tools"] = tools
	}

	return json.Marshal(request)
}

// processBedrockResponse converts a Bedrock API response into our internal session.Message format.
func processBedrockResponse(body []byte, availableTools []tools.Tool) (*session.Message, error) {
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal Bedrock response")
	}

	// Check for error in response
	if errMsg, ok := response["error"]; ok {
		return nil, errors.New("Bedrock API error: %v", errMsg)
	}

	// Extract content from response
	content, ok := response["content"]
	if !ok {
		return &session.Message{Role: "assistant", Content: ""}, nil
	}

	contentArray, ok := content.([]interface{})
	if !ok {
		return nil, errors.New("unexpected content format in Bedrock response")
	}

	var responseContent string
	var toolCalls []session.ToolCall
	toolCallIDCounter := 0

	for _, item := range contentArray {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		itemType, ok := itemMap["type"].(string)
		if !ok {
			continue
		}

		switch itemType {
		case "text":
			if text, ok := itemMap["text"].(string); ok {
				responseContent += text
			}
		case "tool_use":
			// Extract tool call information
			if name, ok := itemMap["name"].(string); ok {
				if input, ok := itemMap["input"].(map[string]interface{}); ok {
					id := fmt.Sprintf("call_%d_%s", toolCallIDCounter, name)
					if toolID, ok := itemMap["id"].(string); ok {
						id = toolID
					}

					toolCall := session.ToolCall{
						ToolCallID: id,
						Name:       name,
						Args:       input,
					}
					toolCalls = append(toolCalls, toolCall)
					toolCallIDCounter++
				}
			}
		}
	}

	return &session.Message{
		Role:      "assistant",
		Content:   responseContent,
		ToolCalls: toolCalls,
	}, nil
}
