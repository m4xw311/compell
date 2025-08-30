package agent

import (
	"context"
	"fmt"

	"github.com/m4xw311/compell/config"
	"github.com/m4xw311/compell/errors"
	"github.com/m4xw311/compell/llm"
	"github.com/m4xw311/compell/session"
	"github.com/m4xw311/compell/tools"
)

type Mode string

const (
	ModeAuto   Mode = "auto"
	ModePrompt Mode = "prompt"
)

// ToolVerbosity defines the level of detail for tool execution logging.
type ToolVerbosity string

const (
	ToolVerbosityNone ToolVerbosity = "none"
	ToolVerbosityInfo ToolVerbosity = "info"
	ToolVerbosityAll  ToolVerbosity = "all"
)

// Agent represents the core agent functionality that can be used by different interfaces (terminal, ACP, etc.)
type Agent struct {
	Config         *config.Config
	Session        *session.Session
	LLMClient      llm.LLMClient
	AvailableTools []tools.Tool
	Mode           Mode
	Verbosity      ToolVerbosity
}

// New creates a new Agent instance with the specified configuration and tools
func New(cfg *config.Config, sess *session.Session, toolset string, mode Mode, client llm.LLMClient, verbosity ToolVerbosity) (*Agent, error) {
	ts, err := cfg.GetToolset(toolset)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get toolset")
	}

	registry := tools.NewToolRegistry(cfg)
	activeTools, err := registry.GetActiveTools(ts)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get active tools")
	}

	return &Agent{
		Config:         cfg,
		Session:        sess,
		LLMClient:      client,
		AvailableTools: activeTools,
		Mode:           mode,
		Verbosity:      verbosity,
	}, nil
}

// ProcessUserInput handles a single user input and returns the assistant's response
// This is the core processing logic that can be used by both terminal and ACP interfaces
func (a *Agent) ProcessUserInput(ctx context.Context, userInput string, callbacks ProcessCallbacks) error {
	userMsg := session.Message{Role: "user", Content: userInput}
	a.Session.AddMessage(userMsg)

	// Main loop: LLM -> Tool -> LLM ...
	for {
		assistantResponse, err := a.LLMClient.Chat(ctx, a.Session.Messages, a.AvailableTools)
		if err != nil {
			return errors.Wrapf(err, "LLM chat failed")
		}

		a.Session.AddMessage(*assistantResponse)

		// If the assistant provided a direct textual response, notify via callback
		if assistantResponse.Content != "" && callbacks.OnAssistantMessage != nil {
			callbacks.OnAssistantMessage(assistantResponse.Content)
		}

		// Break the loop if the LLM provided a final answer (no tool calls)
		if len(assistantResponse.ToolCalls) == 0 {
			// Save session after a complete turn
			if err := a.Session.Save(); err != nil && callbacks.OnWarning != nil {
				callbacks.OnWarning(fmt.Sprintf("failed to save session: %v", err))
			}
			break
		}

		// --- Tool Execution Phase ---
		var toolResultMessages []session.Message

		for _, toolCall := range assistantResponse.ToolCalls {
			// Notify about tool execution if callback is provided
			if callbacks.OnToolCall != nil {
				callbacks.OnToolCall(toolCall)
			}

			// Check if we should execute the tool (for prompt mode)
			shouldExecute := true
			if a.Mode == ModePrompt && callbacks.ShouldExecuteTool != nil {
				shouldExecute = callbacks.ShouldExecuteTool(toolCall)
			}

			var toolResult string
			if !shouldExecute {
				toolResult = "User denied tool execution."
			} else {
				// Execute the tool
				toolResult, err = a.ExecuteToolCall(ctx, toolCall)
				if err != nil {
					// If there was an error during tool execution, format it as a message
					toolResult = fmt.Sprintf("Error executing tool %s: %v", toolCall.Name, err)
				}
			}

			// Notify about tool result if callback is provided
			if callbacks.OnToolResult != nil {
				callbacks.OnToolResult(toolCall, toolResult)
			}

			// Create a message with the tool's output
			toolMsg := session.Message{
				Role:    "tool",
				Content: toolResult,
				ToolCalls: []session.ToolCall{
					{ToolCallID: toolCall.ToolCallID, Name: toolCall.Name},
				},
			}
			toolResultMessages = append(toolResultMessages, toolMsg)
		}

		// Add all tool result messages to the session history at once
		for _, msg := range toolResultMessages {
			a.Session.AddMessage(msg)
		}
		// Continue the loop to send the tool results back to the LLM
	}

	return nil
}

// ExecuteToolCall executes a single tool call and returns the result
func (a *Agent) ExecuteToolCall(ctx context.Context, toolCall session.ToolCall) (string, error) {
	var targetTool tools.Tool
	for _, t := range a.AvailableTools {
		if t.Name() == toolCall.Name {
			targetTool = t
			break
		}
	}

	if targetTool == nil {
		return "", errors.New("tool '%s' not found in the available toolset", toolCall.Name)
	}

	// Execute the tool
	return targetTool.Execute(ctx, toolCall.Args)
}

// ProcessCallbacks defines callbacks for various events during processing
// This allows different interfaces (terminal, ACP) to handle events in their own way
type ProcessCallbacks struct {
	// OnAssistantMessage is called when the assistant produces a text message
	OnAssistantMessage func(message string)

	// OnToolCall is called before a tool is executed
	OnToolCall func(toolCall session.ToolCall)

	// OnToolResult is called after a tool has been executed
	OnToolResult func(toolCall session.ToolCall, result string)

	// ShouldExecuteTool is called in prompt mode to check if a tool should be executed
	// If nil or returns true, the tool will be executed
	ShouldExecuteTool func(toolCall session.ToolCall) bool

	// OnWarning is called for non-fatal warnings (e.g., session save failures)
	OnWarning func(warning string)
}
