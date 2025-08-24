package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/m4xw311/compell/agent"
	"github.com/m4xw311/compell/config"
	"github.com/m4xw311/compell/llm"
	"github.com/m4xw311/compell/session"
)

func main() {
	// Define flags
	modeFlag := flag.String("m", "", "Execution mode: 'auto' or 'prompt'")
	sessionFlag := flag.String("s", "", "Session name to create or use")
	toolsetFlag := flag.String("t", "", "Toolset to use (defaults to 'default')")
	resumeFlag := flag.String("r", "", "Resume a session by name")
	toolVerbosityFlag := flag.String("tool-verbosity", "", "Tool verbosity level: 'none', 'info', or 'all'")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %+v\n", err)
		os.Exit(1)
	}

	var sess *session.Session
	sessionName := *sessionFlag

	if *resumeFlag != "" {
		// Resume session
		sessionName = *resumeFlag
		sess, err = session.Load(sessionName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error resuming session '%s': %+v\n", sessionName, err)
			os.Exit(1)
		}
		fmt.Printf("Resuming session: %s\n", sessionName)
		// Apply session flags if not explicitly overridden by user
		if *modeFlag == "" && sess.Mode != "" {
			*modeFlag = sess.Mode
		}
		if *toolsetFlag == "" && sess.Toolset != "" {
			*toolsetFlag = sess.Toolset
		}
		if *toolVerbosityFlag == "" && sess.ToolVerbosity != "" {
			*toolVerbosityFlag = sess.ToolVerbosity
		}

	} else {
		// Start new session
		if sessionName == "" {
			sessionName = defaultSessionName()
		}
		sess, err = session.New(sessionName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating session '%s': %+v\n", sessionName, err)
			os.Exit(1)
		}
		fmt.Printf("Starting new session: %s\n", sessionName)
	}

	if *modeFlag == "" {
		*modeFlag = "prompt"
	}
	// Seems to work without this. ToDo look into this
	// if *toolsetFlag == "" {
	// 	*toolsetFlag = "default"
	// }
	if *toolVerbosityFlag == "" {
		*toolVerbosityFlag = "none"
	}

	// Update session with current flag values and save
	sess.Mode = *modeFlag
	sess.Toolset = *toolsetFlag
	sess.ToolVerbosity = *toolVerbosityFlag
	if err := sess.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving session '%s': %+v\n", sessionName, err)
		os.Exit(1)
	}

	// Validate mode
	var opMode agent.Mode
	switch *modeFlag {
	case "auto":
		opMode = agent.ModeAuto
	case "prompt":
		opMode = agent.ModePrompt
	default:
		fmt.Fprintf(os.Stderr, "Invalid mode '%s'. Must be 'auto' or 'prompt'.\n", *modeFlag)
		os.Exit(1)
	}

	// Initialize LLM Client
	var client llm.LLMClient
	switch cfg.LLMClient {
	case "gemini":
		client, err = llm.NewGeminiLLMClient(context.Background(), cfg.Model)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing Gemini client: %+v\n", err)
			os.Exit(1)
		}
	case "openai":
		client, err = llm.NewOpenAILLMClient(context.Background(), cfg.Model)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing OpenAI client: %+v\n", err)
			os.Exit(1)
		}
	case "bedrock":
		client, err = llm.NewBedrockLLMClient(context.Background(), cfg.Model)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing Bedrock client: %+v\n", err)
			os.Exit(1)
		}
	default:
		client = &llm.MockLLMClient{}
	}

	// Validate tool verbosity
	var verbosity agent.ToolVerbosity
	switch *toolVerbosityFlag {
	case "none":
		verbosity = agent.ToolVerbosityNone
	case "info":
		verbosity = agent.ToolVerbosityInfo
	case "all":
		verbosity = agent.ToolVerbosityAll
	default:
		fmt.Fprintf(os.Stderr, "Invalid tool verbosity '%s'. Must be 'none', 'info', or 'all'.\n", *toolVerbosityFlag)
		os.Exit(1)
	}

	// Create the agent
	compellAgent, err := agent.New(cfg, sess, *toolsetFlag, opMode, client, verbosity)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing agent: %+v\n", err)
		os.Exit(1)
	}

	// Get initial prompt from remaining arguments
	initialPrompt := strings.Join(flag.Args(), " ")

	// Run the agent
	fmt.Println("Compell is ready. Type your prompt.")
	if err := compellAgent.Run(context.Background(), initialPrompt); err != nil {
		fmt.Fprintf(os.Stderr, "Agent stopped with an error: %+v\n", err)
		os.Exit(1)
	}
}

// newBoolFlag creates a flag.Value that just sets a boolean to true when the flag is present.
// This is used to detect if a flag was explicitly passed, even if its value is the default.
type boolFlag bool

func (b *boolFlag) String() string {
	return fmt.Sprintf("%t", *b)
}

func (b *boolFlag) Set(value string) error {
	*b = true
	return nil
}

func newBoolFlag(b *boolFlag) flag.Value {
	*b = false // Initialize to false
	return b
}

func defaultSessionName() string {
	wd, err := os.Getwd()
	if err != nil {
		wd = "compell"
	}
	dirName := filepath.Base(wd)
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	return fmt.Sprintf("%s_%s", dirName, timestamp)
}
