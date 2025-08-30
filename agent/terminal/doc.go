// Package terminal implements the command-line interface (CLI) mode for the Compell agent.
//
// This package provides an interactive terminal-based user interface where users can
// communicate with the agent through text prompts and receive responses directly in
// the terminal. It handles user input, displays agent responses, manages tool execution
// confirmations (in prompt mode), and provides appropriate verbosity levels for tool
// execution output.
//
// The terminal package is one of the two main interaction modes for Compell:
//   - Terminal mode: Interactive CLI for direct user interaction
//   - ACP mode: JSON-RPC based protocol for IDE integration
//
// # Usage
//
// To use the terminal interface, create an agent instance and pass it to the terminal:
//
//	agent, err := agent.New(cfg, session, toolset, mode, llmClient, verbosity)
//	if err != nil {
//	    // handle error
//	}
//
//	term := terminal.New(agent)
//	err = term.Run(ctx, initialPrompt)
//
// # Features
//
//   - Interactive prompt-based conversation with the agent
//   - Support for initial prompts from command-line arguments
//   - Tool execution confirmation in prompt mode
//   - Configurable verbosity levels for tool execution output
//   - Session management with conversation history
//   - Exit commands (/quit, /exit) for graceful termination
//
// # Modes
//
// The terminal respects the agent's operation mode:
//
//   - Auto mode: Tools are executed automatically without user confirmation
//   - Prompt mode: User is prompted for confirmation before each tool execution
//
// # Verbosity Levels
//
// The terminal supports different verbosity levels for tool execution:
//
//   - None: No tool execution information is displayed
//   - Info: Tool names are displayed when called
//   - All: Tool names, arguments, and results are displayed
package terminal
