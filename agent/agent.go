package agent

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

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

type Agent struct {
	Config         *config.Config
	Session        *session.Session
	LLMClient      llm.LLMClient
	AvailableTools []tools.Tool
	Mode           Mode
	Verbosity      ToolVerbosity
}

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

func (a *Agent) Run(ctx context.Context, initialPrompt string) error {
	// If there's an initial prompt from the command line, use it first.
	if initialPrompt != "" {
		if err := a.processTurn(ctx, initialPrompt); err != nil {
			return err
		}
	}

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("You: ")
		if !scanner.Scan() {
			// EOF or read error ends the session
			break
		}

		userInput := strings.TrimSpace(scanner.Text())
		if userInput == "" {
			continue
		}

		// Exit commands
		if userInput == "/quit" || userInput == "/exit" {
			break
		}

		if err := a.processTurn(ctx, userInput); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func (a *Agent) processTurn(ctx context.Context, userInput string) error {
	userMsg := session.Message{Role: "user", Content: userInput}
	a.Session.AddMessage(userMsg)

	// Main loop: LLM -> Tool -> LLM ...
	for {
		assistantResponse, err := a.LLMClient.Chat(ctx, a.Session.Messages, a.AvailableTools)
		if err != nil {
			return errors.Wrapf(err, "LLM chat failed")
		}

		a.Session.AddMessage(*assistantResponse)

		// If the assistant provided a direct textual response, print it.
		if assistantResponse.Content != "" {
			fmt.Printf("Compell: %s\n", assistantResponse.Content)
		}

		// Break the loop if the LLM provided a final answer (no tool calls)
		if len(assistantResponse.ToolCalls) == 0 {
			// Save session after a complete turn
			if err := a.Session.Save(); err != nil {
				fmt.Printf("Warning: failed to save session: %v\n", err)
			}
			break
		}

		// --- Tool Execution Phase ---

		var toolResultMessages []session.Message

		for _, toolCall := range assistantResponse.ToolCalls {
			toolResult, err := a.executeToolCall(ctx, toolCall)
			if err != nil {
				// If there was an error during tool execution (e.g., tool not found),
				// format it as a message to be sent back to the LLM.
				toolResult = fmt.Sprintf("Error executing tool %s: %v", toolCall.Name, err)
			}

			if a.Verbosity == ToolVerbosityAll {
				fmt.Printf("Tool `%s` output: %s\n", toolCall.Name, toolResult)
			}

			// Create a message with the tool's output.
			toolMsg := session.Message{
				Role:    "tool",
				Content: toolResult,
				ToolCalls: []session.ToolCall{
					{ToolCallID: toolCall.ToolCallID, Name: toolCall.Name},
				},
			}
			toolResultMessages = append(toolResultMessages, toolMsg)
		}

		// Add all tool result messages to the session history at once.
		for _, msg := range toolResultMessages {
			a.Session.AddMessage(msg)
		}
		// Continue the loop to send the tool results back to the LLM.
	}

	return nil
}

func (a *Agent) executeToolCall(ctx context.Context, toolCall session.ToolCall) (string, error) {
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

	if a.Verbosity == ToolVerbosityAll {
		fmt.Printf("Compell wants to call tool `%s` with args: %v\n", toolCall.Name, toolCall.Args)
	} else if a.Verbosity == ToolVerbosityInfo {
		fmt.Printf("Compell wants to call tool `%s`\n", toolCall.Name)
	}

	// In prompt mode, ask for user confirmation.
	if a.Mode == ModePrompt {
		fmt.Print("Do you want to allow this? (y/n): ")
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		if strings.TrimSpace(strings.ToLower(answer)) != "y" {
			return "User denied tool execution.", nil
		}
	}

	// Execute the tool.
	return targetTool.Execute(ctx, toolCall.Args)
}
