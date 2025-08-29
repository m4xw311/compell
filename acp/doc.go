// Package acp implements the Agent Client Protocol (ACP) support for Compell.
// This allows Compell to integrate with code editors like Zed by communicating
// using JSON-RPC over stdio with Content-Length framing.
//
// The implementation supports the following ACP methods:
// - initialize: Initializes the agent and returns capabilities
// - session/new: Creates a new session
// - session/prompt: Processes a prompt and returns the result
//
// The implementation sends the following notifications:
// - session/update: Streams agent responses back to the client with agent_message_chunk updates
package acp
