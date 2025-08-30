// Package acp implements the Agent Client Protocol (ACP) support for Compell.
// This allows Compell to integrate with code editors like Zed by communicating
// using JSON-RPC over stdio.
//
// The implementation supports the following ACP methods:
// - initialize: Initializes the agent and returns capabilities
// - session/new: Creates a new session
// - session/load: Loads an existing session and replays conversation history
// - session/prompt: Processes a prompt and returns the result
//
// The implementation sends the following notifications:
// - session/update: Streams agent responses back to the client with:
//   - agent_message_chunk: Text content from the agent
//   - tool_call: Tool execution requests from the agent
//   - tool_result: Results from executed tools
//   - user_message_chunk: Replayed user messages when loading sessions
//
// Note: Messages are newline-delimited JSON objects. Content-Length framing
// should not be used as it is not necessary when communicating over stdio.
// Further details on ACP can be found in project/features/acp/ACP.md
package acp
