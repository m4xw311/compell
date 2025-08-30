// Package agent provides the core agent functionality for the Compell system.
//
// This package contains the common code and abstractions that are shared between
// different interaction modes (terminal CLI and ACP server). It defines the core
// Agent type and the processing logic for handling user input, LLM interactions,
// and tool executions.
//
// # Architecture
//
// The agent package is organized into three main components:
//
//   - Core agent (this package): Contains the shared Agent type and processing logic
//   - Terminal subpackage (agent/terminal): Implements the CLI interaction mode
//   - ACP subpackage (agent/acp): Implements the Agent Client Protocol server for IDE integration
//
// # Core Functionality
//
// The Agent type provides:
//
//   - Configuration management for LLM clients and toolsets
//   - Session management for conversation history
//   - Tool discovery and execution
//   - Processing loop for LLM interactions and tool calls
//   - Callback-based architecture for different interaction modes
//
// # Usage
//
// To create and use an agent:
//
//	// Create an agent with configuration
//	agent, err := agent.New(cfg, session, toolset, mode, llmClient, verbosity)
//	if err != nil {
//	    // handle error
//	}
//
//	// Define callbacks for your interaction mode
//	callbacks := agent.ProcessCallbacks{
//	    OnAssistantMessage: func(message string) {
//	        // Handle assistant responses
//	    },
//	    OnToolCall: func(toolCall session.ToolCall) {
//	        // Handle tool execution requests
//	    },
//	    OnToolResult: func(toolCall session.ToolCall, result string) {
//	        // Handle tool execution results
//	    },
//	    ShouldExecuteTool: func(toolCall session.ToolCall) bool {
//	        // Determine if a tool should be executed (for prompt mode)
//	        return true
//	    },
//	    OnWarning: func(warning string) {
//	        // Handle non-fatal warnings
//	    },
//	}
//
//	// Process user input
//	err = agent.ProcessUserInput(ctx, "user message", callbacks)
//
// # Modes
//
// The agent supports two operation modes:
//
//   - ModeAuto: Tools are executed automatically without confirmation
//   - ModePrompt: Tool execution requires confirmation (handled via callbacks)
//
// # Tool Verbosity
//
// Tool execution verbosity can be configured at three levels:
//
//   - ToolVerbosityNone: No tool execution details are shown
//   - ToolVerbosityInfo: Basic tool execution information is shown
//   - ToolVerbosityAll: Detailed tool execution information including arguments and results
//
// # Callbacks
//
// The ProcessCallbacks structure allows different interaction modes to customize
// how agent events are handled. This design enables the same core processing logic
// to be used by both the terminal CLI and the ACP server, while allowing each to
// handle events in their own way (e.g., printing to stdout vs. sending JSON-RPC
// notifications).
//
// # Subpackages
//
// agent/terminal: Provides an interactive command-line interface for direct user
// interaction with the agent. Features include prompt-based conversations, tool
// execution confirmations, and configurable verbosity.
//
// agent/acp: Implements the Agent Client Protocol server for IDE integration.
// Provides JSON-RPC based communication over stdio, session management, and
// real-time updates via notifications.
package agent
