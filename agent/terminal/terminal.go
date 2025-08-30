package terminal

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/m4xw311/compell/agent"
	"github.com/m4xw311/compell/session"
)

// Terminal handles the terminal/CLI interaction mode for the agent
type Terminal struct {
	agent *agent.Agent
}

// New creates a new Terminal instance
func New(a *agent.Agent) *Terminal {
	return &Terminal{
		agent: a,
	}
}

// Run starts the interactive terminal session
func (t *Terminal) Run(ctx context.Context, initialPrompt string) error {
	// If there's an initial prompt from the command line, use it first
	if initialPrompt != "" {
		if err := t.processTurn(ctx, initialPrompt); err != nil {
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

		if err := t.processTurn(ctx, userInput); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

// processTurn handles a single user input turn
func (t *Terminal) processTurn(ctx context.Context, userInput string) error {
	// Create callbacks for terminal-specific behavior
	callbacks := agent.ProcessCallbacks{
		OnAssistantMessage: func(message string) {
			fmt.Printf("Compell: %s\n", message)
		},
		OnToolCall: func(toolCall session.ToolCall) {
			// Display tool call information based on verbosity
			if t.agent.Verbosity == agent.ToolVerbosityAll {
				fmt.Printf("Compell wants to call tool `%s` with args: %v\n", toolCall.Name, toolCall.Args)
			} else if t.agent.Verbosity == agent.ToolVerbosityInfo {
				fmt.Printf("Compell wants to call tool `%s`\n", toolCall.Name)
			}
		},
		OnToolResult: func(toolCall session.ToolCall, result string) {
			// Display tool result if verbosity is set to all
			if t.agent.Verbosity == agent.ToolVerbosityAll {
				fmt.Printf("Tool `%s` output: %s\n", toolCall.Name, result)
			}
		},
		ShouldExecuteTool: func(toolCall session.ToolCall) bool {
			// In prompt mode, ask for user confirmation
			if t.agent.Mode == agent.ModePrompt {
				fmt.Print("Do you want to allow this? (y/n): ")
				reader := bufio.NewReader(os.Stdin)
				answer, _ := reader.ReadString('\n')
				return strings.TrimSpace(strings.ToLower(answer)) == "y"
			}
			// In auto mode, always execute
			return true
		},
		OnWarning: func(warning string) {
			fmt.Printf("Warning: %s\n", warning)
		},
	}

	return t.agent.ProcessUserInput(ctx, userInput, callbacks)
}
