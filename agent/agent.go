package agent

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/m4xw311/compell/config"
	"github.com/m4xw311/compell/llm"
	"github.com/m4xw311/compell/session"
	"github.com/m4xw311/compell/tools"
)

type Mode string

const (
	ModeAuto   Mode = "auto"
	ModePrompt Mode = "prompt"
)

type Agent struct {
	Config         *config.Config
	Session        *session.Session
	LLMClient      llm.LLMClient
	AvailableTools []tools.Tool
	Mode           Mode
}

func New(cfg *config.Config, sess *session.Session, toolset string, mode Mode, client llm.LLMClient) (*Agent, error) {
	ts, err := cfg.GetToolset(toolset)
	if err != nil {
		return nil, err
	}

	registry := tools.NewToolRegistry(cfg)
	activeTools, err := registry.GetActiveTools(ts)
	if err != nil {
		return nil, err
	}

	return &Agent{
		Config:         cfg,
		Session:        sess,
		LLMClient:      client,
		AvailableTools: activeTools,
		Mode:           mode,
	}, nil
}

func (a *Agent) Run(ctx context.Context, initialPrompt string) error {
	reader := bufio.NewReader(os.Stdin)

	// If there's an initial prompt from the command line, use it first.
	if initialPrompt != "" {
		if err := a.processTurn(ctx, initialPrompt); err != nil {
			return err
		}
	}

	for {
		fmt.Print("You: ")
		userInput, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		userInput = strings.TrimSpace(userInput)
		if userInput == "" {
			continue
		}

		if err := a.processTurn(ctx, userInput); err != nil {
			fmt.Printf("Error: %v\n", err)
			// Decide if you want to continue or exit on error
		}
	}
}

func (a *Agent) processTurn(ctx context.Context, userInput string) error {
	userMsg := session.Message{Role: "user", Content: userInput}
	a.Session.AddMessage(userMsg)

	// Main loop: LLM -> Tool -> LLM ...
	for {
		assistantResponse, err := a.LLMClient.Chat(ctx, a.Session.Messages, a.AvailableTools)
		if err != nil {
			return fmt.Errorf("LLM chat failed: %w", err)
		}

		// A real implementation would check for `assistantResponse.ToolCalls`
		// and execute them here, possibly prompting the user if in `prompt` mode.
		// For this example, we assume the LLM just returns text.

		a.Session.AddMessage(*assistantResponse)
		fmt.Printf("Compell: %s\n", assistantResponse.Content)

		// Save session after each turn
		if err := a.Session.Save(); err != nil {
			fmt.Printf("Warning: failed to save session: %v\n", err)
		}

		// Break the loop if the LLM provided a final answer (no tool calls)
		// if len(assistantResponse.ToolCalls) == 0 {
		break
		// }

		// ... tool execution logic would go here ...
	}

	return nil
}
