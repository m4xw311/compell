# Project Overview
Compell - the no fluff no bs coding assistant.
- A command line coding assistant
- Flexible LLM integrations
- Customizable tools
- Support for web interface over websocket - Early prototype
- Support for Agent Client Protocol (ACP) - Implemented

# Implementation Details

Compell is structured as a Go application with several key components:

## Core Components

- **`cmd/compell`** - Main command-line interface containing the entry point (`main.go`) that handles argument parsing, session management, and agent initialization
- **`agent/`** - Contains the core agent logic that processes user input, communicates with LLMs, and executes tools
- **`llm/`** - LLM client implementations for various providers (OpenAI, Gemini, Anthropic/Bedrock) with a common interface
- **`tools/`** - Implementation of all available tools including filesystem operations and command execution
- **`session/` - Session management for persisting conversation history and state
- **`config/`** - Configuration loading and management from YAML files
- **`errors/`** - Custom error handling utilities

## Specialized Components

- **`acp/`** - Agent Client Protocol (ACP) integration for connecting to code editors like Zed
- **`tools/mcp/`** - Multi-Client Protocol (MCP) integration for connecting to external tool servers
- **`cmd/ws_bridge`** - WebSocket bridge for web interface support
- **`web/`** - Basic web interface files

## Key Files

- **`main.go`** - Entry point that handles command-line arguments, loads configuration, initializes sessions, and starts the agent
- **`agent/agent.go`** - Core agent implementation that manages the interaction loop between user, LLM, and tools
- **`llm/client.go`** - Common interface for all LLM clients
- **`tools/tools.go`** - Tool registry and management system
- **`config/config.go`** - Configuration loading from user and project level YAML files
- **`session/session.go`** - Session persistence using JSON files

## Tool Architecture

Tools are implemented as interfaces with standardized execution methods. The tool registry system allows for both built-in tools (filesystem operations, command execution) and external MCP tools. Tools can be grouped into toolsets defined in the configuration file.

## Configuration

Configuration is handled through YAML files that can be specified at both user level (`~/.compell/config.yaml`) and project level (`./.compell/config.yaml`). Configuration includes LLM provider settings, toolset definitions, filesystem access restrictions, and command whitelists.

## Agent Client Protocol (ACP) Support

Compell supports the Agent Client Protocol (ACP), which allows it to integrate with code editors like Zed. To enable ACP mode, use the `--acp` flag when running Compell. In ACP mode, Compell communicates using JSON-RPC over stdio instead of the regular CLI interaction.

ACP mode supports the following methods:
- `initialize` - Initializes the agent and returns capabilities
- `session/new` - Creates a new session
- `session/prompt` - Processes a prompt and returns the result
- `session/update` notifications - Streams agent responses back to the client

When running in ACP mode, Compell implements proper JSON-RPC framing with Content-Length headers.