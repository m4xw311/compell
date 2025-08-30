package acp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/m4xw311/compell/agent"
	"github.com/m4xw311/compell/session"
)

// Run starts the Agent Client Protocol server over stdio using JSON-RPC
// It implements a minimal subset of ACP:
// - initialize
// - session/new
// - session/load
// - session/prompt (emits session/update notifications with agent_message_chunk, tool_call, and tool_result)
// Notes:
// - This implementation intentionally avoids writing anything to stdout except JSON-RPC messages.
// - Any debug or informational logs should go to trace file if needed.
// - Messages are newline-delimited JSON objects rather than using Content-Length framing.
func Run(ctx context.Context, compellAgent *agent.Agent, in *bufio.Reader, out *bufio.Writer, traceFlag *bool) error {
	var traceFile *os.File
	trace := func(msg string) {} // Do nothing by default
	if *traceFlag == true {
		traceFile, _ = os.OpenFile("acp.trace", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		defer traceFile.Close()
		// Write trace messages to the file
		trace = func(msg string) {
			if traceFile != nil {
				fmt.Fprintf(traceFile, "[%s] %s\n", time.Now().Format("15:04:05.000"), msg)
			}
		}
	}

	trace("Run: starting ACP server")
	server := &acpServer{
		ctx:          ctx,
		agent:        compellAgent,
		sessions:     make(map[string]*session.Session),
		sessionIDSeq: 0,
		StdinReader:  in,
		StdoutWriter: out,
		writeLock:    &sync.Mutex{},
		trace:        trace,
	}

	// Main read loop
	for {
		trace("Run: entering read loop")
		// Read a framed JSON-RPC message from stdin
		payload, err := server.readFramedMessage()
		if err != nil {
			if err == io.EOF {
				trace("Run: EOF received, exiting")
				return nil
			}
			// If framing is broken, there isn't a safe way to continue.
			trace(fmt.Sprintf("Run: read error: %v", err))
			return fmt.Errorf("ACP: read error: %w", err)
		}
		if len(payload) == 0 {
			trace("Run: empty payload, continuing")
			// Nothing to process, continue
			continue
		}

		trace(fmt.Sprintf("Run: received payload: %s", string(payload)))
		// Parse request
		var req jsonrpcRequest
		if err := json.Unmarshal(payload, &req); err != nil {
			trace(fmt.Sprintf("Run: JSON parse error: %v", err))
			// Return JSON-RPC parse error
			_ = server.writeResponseError(nil, -32700, "Parse error", nil)
			continue
		}

		trace(fmt.Sprintf("Run: dispatching method: %s with ID: %v", req.Method, req.ID))
		// Dispatch on method
		switch req.Method {
		case "initialize":
			trace("Run: calling handleInitialize")
			server.handleInitialize(&req)
		case "session/new":
			trace("Run: calling handleSessionNew")
			server.handleSessionNew(&req)
		case "session/load":
			trace("Run: calling handleSessionLoad")
			server.handleSessionLoad(&req)
		case "session/prompt":
			trace("Run: calling handleSessionPrompt")
			server.handleSessionPrompt(&req)
		default:
			trace("Run: method not found")
			// Method not found
			_ = server.writeResponseError(req.ID, -32601, "Method not found", nil)
		}
	}
}

// ---- Minimal ACP handling types ----

// jsonrpcRequest represents a JSON-RPC 2.0 request message
type jsonrpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id,omitempty"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// jsonrpcResponse represents a JSON-RPC 2.0 response message
type jsonrpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonrpcError   `json:"error,omitempty"`
}

// jsonrpcError represents a JSON-RPC 2.0 error object
type jsonrpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// ---- acpServer ----

// acpServer represents the state of an ACP server instance
// It manages sessions, handles requests, and communicates with the client over stdio
type acpServer struct {
	ctx          context.Context
	agent        *agent.Agent
	sessions     map[string]*session.Session
	sessionsLock sync.Mutex
	sessionIDSeq int64

	StdinReader  *bufio.Reader
	StdoutWriter *bufio.Writer
	writeLock    *sync.Mutex
	trace        func(string)
}

// readFramedMessage reads a single JSON-RPC payload
func (s *acpServer) readFramedMessage() ([]byte, error) {
	s.trace("readFramedMessage: starting")
	// JSON-RPC requests and responses are newline-delimited JSONs.
	line, _, err := s.StdinReader.ReadLine()
	if err != nil {
		s.trace(fmt.Sprintf("readFramedMessage: error reading message: %v", err))
		return nil, err
	}

	s.trace(fmt.Sprintf("readFramedMessage: successfully read direct JSON message of length %d: %s", len(line), string(line)))
	return line, nil
}

// writeFramedJSON serializes and writes a JSON-RPC message to stdout
// It handles the newline-delimited JSON formatting required by the ACP protocol
func (s *acpServer) writeFramedJSON(obj any) error {
	s.trace("writeFramedJSON: starting")
	data, err := json.Marshal(obj)
	if err != nil {
		s.trace(fmt.Sprintf("writeFramedJSON: marshal error: %v", err))
		return fmt.Errorf("failed to serialize JSON-RPC message: %w", err)
	}
	s.trace(fmt.Sprintf("writeFramedJSON: %s", string(data)))

	s.writeLock.Lock()
	defer s.writeLock.Unlock()
	if _, err := s.StdoutWriter.Write(data); err != nil {
		s.trace(fmt.Sprintf("writeFramedJSON: write error: %v", err))
		return err
	}
	// JSON-RPC requests and responses are newline-delimited JSONs.
	// Write newline to stdout to inform client that message is complete
	if _, err := s.StdoutWriter.WriteString("\n"); err != nil {
		s.trace(fmt.Sprintf("writeFramedJSON: write error: %v", err))
		return err
	}
	err = s.StdoutWriter.Flush()
	if err != nil {
		s.trace(fmt.Sprintf("writeFramedJSON: flush error: %v", err))
		return err
	}
	s.trace("writeFramedJSON: successfully wrote message")
	return nil
}

// writeResponseOK sends a successful JSON-RPC response with the given result
func (s *acpServer) writeResponseOK(id any, result json.RawMessage) error {
	s.trace("writeResponseOK: starting")
	resp := jsonrpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	return s.writeFramedJSON(resp)
}

// writeResponseError sends a JSON-RPC error response with the specified error code and message
func (s *acpServer) writeResponseError(id any, code int, msg string, data any) error {
	s.trace(fmt.Sprintf("writeResponseError: code=%d, msg=%s, data=%+v", code, msg, data))
	resp := jsonrpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &jsonrpcError{
			Code:    code,
			Message: msg,
			Data:    data,
		},
	}
	return s.writeFramedJSON(resp)
}

// writeNotification sends a JSON-RPC notification (request without an ID)
func (s *acpServer) writeNotification(method string, params any) error {
	s.trace(fmt.Sprintf("writeNotification: method=%s, params=%+v", method, params))
	// Notifications have no id
	msg := map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
	}
	return s.writeFramedJSON(msg)
}

// ---- Handlers ----

// handleInitialize processes the initialize request from the ACP client
// It returns the protocol version and agent capabilities including:
// - Support for session loading
// - Prompt capabilities (currently no support for audio, embedded context, or image)
func (s *acpServer) handleInitialize(req *jsonrpcRequest) {
	s.trace("handleInitialize: starting")
	// initParams represents the parameters for the initialize request
	type initParams struct {
		ProtocolVersion int             `json:"protocolVersion"`
		ClientCaps      json.RawMessage `json:"clientCapabilities,omitempty"`
	}

	var p initParams
	b, err := json.Marshal(req.Params)
	if err != nil {
		s.trace(fmt.Sprintf("handleInitialize: json marshal error : %v", err))
	}
	err = json.Unmarshal(b, &p)
	if err != nil {
		s.trace(fmt.Sprintf("handleInitialize: json unmarshal error : %v", err))
	}

	// Minimal: we support v1
	resp := map[string]any{
		"protocolVersion": 1,
		"agentCapabilities": map[string]any{
			"loadSession": true,
			"promptCapabilities": map[string]bool{
				"audio":           false,
				"embeddedContext": false,
				"image":           false,
			},
		},
		"authMethods": []any{},
	}
	respBytes, err := json.Marshal(resp)
	if err != nil {
		s.trace(fmt.Sprintf("Error marshalling map: %v", err))
	}
	rawResp := json.RawMessage(respBytes)

	s.trace(fmt.Sprintf("handleInitialize: sending response: %s", string(respBytes)))
	_ = s.writeResponseOK(req.ID, rawResp)
}

// handleSessionNew creates a new session with a unique ID
// It initializes the session with agent configuration metadata and stores it
// Returns the session ID to the client
func (s *acpServer) handleSessionNew(req *jsonrpcRequest) {
	s.trace("handleSessionNew: starting")
	// sessionNewParams represents the parameters for creating a new session
	type sessionNewParams struct {
		Cwd        string          `json:"cwd"`
		McpServers json.RawMessage `json:"mcpServers"`
	}
	var p sessionNewParams
	b, err := json.Marshal(req.Params)
	if err != nil {
		s.trace(fmt.Sprintf("handleInitialize: err : %v", err))
	}
	err = json.Unmarshal(b, &p)
	if err != nil {
		s.trace(fmt.Sprintf("handleInitialize: err : %v", err))
	}

	// Create a new session ID and session object
	sid := s.nextSessionID()
	s.trace(fmt.Sprintf("handleSessionNew: created session ID: %s", sid))

	// Create a new session with the session ID as its name
	sess, err := session.New(sid)
	if err != nil {
		s.trace(fmt.Sprintf("handleSessionNew: failed to create session: %v", err))
		_ = s.writeResponseError(req.ID, -32603, "Internal error", fmt.Sprintf("failed to create session: %v", err))
		return
	}

	// Store session metadata from the agent configuration
	sess.Mode = string(s.agent.Session.Mode)
	sess.Toolset = s.agent.Session.Toolset
	sess.ToolVerbosity = string(s.agent.Session.ToolVerbosity)
	sess.Acp = s.agent.Session.Acp

	s.sessionsLock.Lock()
	s.sessions[sid] = sess
	s.sessionsLock.Unlock()

	resp := map[string]any{
		"sessionId": sid,
	}
	respBytes, err := json.Marshal(resp)
	if err != nil {
		s.trace(fmt.Sprintf("Error marshalling map: %v", err))
	}
	rawResp := json.RawMessage(respBytes)
	s.trace(fmt.Sprintf("handleSessionNew: sending response: %s", string(respBytes)))
	_ = s.writeResponseOK(req.ID, rawResp)
}

// handleSessionLoad loads an existing session from disk and replays the conversation history
// It sends session/update notifications to replay:
// - user_message_chunk for user messages
// - agent_message_chunk for assistant text responses
// - tool_call for tool execution requests
// - tool_result for tool execution results
// Returns null when replay is complete
func (s *acpServer) handleSessionLoad(req *jsonrpcRequest) {
	s.trace("handleSessionLoad: starting")
	// sessionLoadParams represents the parameters for loading an existing session
	type sessionLoadParams struct {
		SessionID  string          `json:"sessionId"`
		Cwd        string          `json:"cwd"`
		McpServers json.RawMessage `json:"mcpServers"`
	}
	var p sessionLoadParams
	b, err := json.Marshal(req.Params)
	if err != nil {
		s.trace(fmt.Sprintf("handleSessionLoad: marshal error: %v", err))
		_ = s.writeResponseError(req.ID, -32603, "Internal error", fmt.Sprintf("marshal error: %v", err))
		return
	}
	err = json.Unmarshal(b, &p)
	if err != nil {
		s.trace(fmt.Sprintf("handleSessionLoad: unmarshal error: %v", err))
		_ = s.writeResponseError(req.ID, -32603, "Internal error", fmt.Sprintf("unmarshal error: %v", err))
		return
	}

	// Load the session from disk
	s.trace(fmt.Sprintf("handleSessionLoad: loading session: %s", p.SessionID))
	sess, err := session.Load(p.SessionID)
	if err != nil {
		s.trace(fmt.Sprintf("handleSessionLoad: failed to load session: %v", err))
		_ = s.writeResponseError(req.ID, -32602, "Invalid params", fmt.Sprintf("session not found: %v", err))
		return
	}

	// Store the loaded session in memory
	s.sessionsLock.Lock()
	s.sessions[p.SessionID] = sess
	s.sessionsLock.Unlock()

	// Replay the conversation history to the client
	s.trace(fmt.Sprintf("handleSessionLoad: replaying %d messages", len(sess.Messages)))
	for _, msg := range sess.Messages {
		switch msg.Role {
		case "user":
			// Send user message chunk notification
			s.trace(fmt.Sprintf("handleSessionLoad: replaying user message: %s", msg.Content))
			_ = s.writeNotification("session/update", map[string]any{
				"sessionId": p.SessionID,
				"update": map[string]any{
					"sessionUpdate": "user_message_chunk",
					"content": map[string]any{
						"type": "text",
						"text": msg.Content,
					},
				},
			})
		case "assistant":
			// Send agent message chunk notification
			if msg.Content != "" {
				s.trace(fmt.Sprintf("handleSessionLoad: replaying assistant message: %s", msg.Content))
				_ = s.writeNotification("session/update", map[string]any{
					"sessionId": p.SessionID,
					"update": map[string]any{
						"sessionUpdate": "agent_message_chunk",
						"content": map[string]any{
							"type": "text",
							"text": msg.Content,
						},
					},
				})
			}
			// Also replay tool calls if any
			for _, tc := range msg.ToolCalls {
				s.trace(fmt.Sprintf("handleSessionLoad: replaying tool call: %s", tc.Name))
				_ = s.sendToolCallNotification(p.SessionID, tc)
			}
		case "tool":
			// Tool results are part of the conversation but typically not shown directly
			// They're associated with the previous tool call
			s.trace("handleSessionLoad: replaying tool result")
			// Find the tool call ID from the message
			if len(msg.ToolCalls) > 0 {
				_ = s.sendToolResultNotification(p.SessionID, msg.ToolCalls[0].ToolCallID, msg.Content)
			}
		}
	}

	// Send response indicating load is complete
	s.trace("handleSessionLoad: sending response")
	_ = s.writeResponseOK(req.ID, json.RawMessage("null"))
}

// contentBlock represents a content block in ACP prompt requests.
// For this minimal implementation, we only handle text blocks.
type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	// ResourceLink fields
	URI         string `json:"uri,omitempty"`
	Name        string `json:"name,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Size        *int64 `json:"size,omitempty"`
}

// handleSessionPrompt processes a prompt request for a session
// It handles the full LLM tool calling loop:
// 1. Appends user message to session history
// 2. Calls LLM with current history
// 3. Streams agent responses via agent_message_chunk notifications
// 4. Executes any tool calls and sends tool_call/tool_result notifications
// 5. Continues loop until LLM indicates completion
// 6. Saves session to disk and returns stopReason: end_turn
func (s *acpServer) handleSessionPrompt(req *jsonrpcRequest) {
	s.trace("handleSessionPrompt: starting")
	// promptParams represents the parameters for processing a prompt
	type promptParams struct {
		SessionID string         `json:"sessionId"`
		Prompt    []contentBlock `json:"prompt"`
	}

	var p promptParams
	b, err := json.Marshal(req.Params)
	if err != nil {
		s.trace(fmt.Sprintf("handleSessionPrompt: marshal error: %v", err))
		_ = s.writeResponseError(req.ID, -32603, "Internal error", fmt.Sprintf("marshal error: %v", err))
		return
	}
	err = json.Unmarshal(b, &p)
	if err != nil {
		s.trace(fmt.Sprintf("handleSessionPrompt: unmarshal error: %v", err))
		_ = s.writeResponseError(req.ID, -32603, "Internal error", fmt.Sprintf("unmarshal error: %v", err))
		return
	}

	// Find session
	s.trace(fmt.Sprintf("handleSessionPrompt: looking up session: %s", p.SessionID))
	s.sessionsLock.Lock()
	sess, ok := s.sessions[p.SessionID]
	s.sessionsLock.Unlock()
	if !ok {
		s.trace("handleSessionPrompt: unknown sessionId")
		_ = s.writeResponseError(req.ID, -32602, "Invalid params", "unknown sessionId")
		return
	}

	// Extract user text from prompt content blocks
	s.trace(fmt.Sprintf("handleSessionPrompt: received %d content blocks", len(p.Prompt)))
	for i, block := range p.Prompt {
		switch block.Type {
		case "text":
			s.trace(fmt.Sprintf("  Block %d: type=text, text=%q", i, block.Text))
		case "resource_link":
			s.trace(fmt.Sprintf("  Block %d: type=resource_link, uri=%s, name=%s, mimeType=%s, title=%s, description=%s, size=%v",
				i, block.URI, block.Name, block.MimeType, block.Title, block.Description, block.Size))
		default:
			s.trace(fmt.Sprintf("  Block %d: type=%s (unsupported)", i, block.Type))
		}
	}
	userText := extractUserText(p.Prompt)
	s.trace(fmt.Sprintf("handleSessionPrompt: extracted user text: %s", userText))

	// Note: ProcessUserInput will add the user message to the session
	// so we don't need to do it here to avoid duplication

	// Create callbacks for ACP-specific behavior
	callbacks := agent.ProcessCallbacks{
		OnAssistantMessage: func(message string) {
			s.trace(fmt.Sprintf("handleSessionPrompt: sending agent message chunk with content: %s", message))
			_ = s.sendAgentMessageChunk(p.SessionID, message)
		},
		OnToolCall: func(toolCall session.ToolCall) {
			s.trace(fmt.Sprintf("handleSessionPrompt: executing tool call: %s with args: %v", toolCall.Name, toolCall.Args))
			// Send tool_call notification
			_ = s.sendToolCallNotification(p.SessionID, toolCall)
		},
		OnToolResult: func(toolCall session.ToolCall, result string) {
			// Send tool_result notification
			_ = s.sendToolResultNotification(p.SessionID, toolCall.ToolCallID, result)
		},
		ShouldExecuteTool: func(toolCall session.ToolCall) bool {
			// In ACP mode, always execute tools (no user prompt needed)
			return true
		},
		OnWarning: func(warning string) {
			s.trace(fmt.Sprintf("handleSessionPrompt: warning - %s", warning))
		},
	}

	// Process the user input using the agent
	s.trace("handleSessionPrompt: processing user input with agent")
	s.agent.Session = sess // Update agent's session to use the ACP session
	if err := s.agent.ProcessUserInput(s.ctx, userText, callbacks); err != nil {
		s.trace(fmt.Sprintf("handleSessionPrompt: error processing user input: %v", err))
		_ = s.writeResponseError(req.ID, -32603, "Internal error", fmt.Sprintf("error processing user input: %v", err))
		return
	}

	// Respond with stopReason: end_turn
	resp := map[string]any{
		"stopReason": "end_turn",
	}
	respBytes, err := json.Marshal(resp)
	if err != nil {
		s.trace(fmt.Sprintf("Error marshalling map: %v", err))
	}
	rawResp := json.RawMessage(respBytes)
	s.trace(fmt.Sprintf("handleSessionPrompt: sending response: %s", string(respBytes)))
	_ = s.writeResponseOK(req.ID, rawResp)
}

// sendToolCallNotification emits a session/update notification for a tool call
// This informs the client that the agent wants to execute a tool with specific arguments
func (s *acpServer) sendToolCallNotification(sessionID string, toolCall session.ToolCall) error {
	s.trace(fmt.Sprintf("sendToolCallNotification: session=%s, tool=%s", sessionID, toolCall.Name))
	notification := map[string]any{
		"sessionId": sessionID,
		"update": map[string]any{
			"sessionUpdate": "tool_call",
			"toolCall": map[string]any{
				"id":   toolCall.ToolCallID,
				"name": toolCall.Name,
				"args": toolCall.Args,
			},
		},
	}
	return s.writeNotification("session/update", notification)
}

// sendToolResultNotification emits a session/update notification for a tool result
// This informs the client of the result from executing a tool call
func (s *acpServer) sendToolResultNotification(sessionID, toolCallID, result string) error {
	s.trace(fmt.Sprintf("sendToolResultNotification: session=%s, toolCallID=%s", sessionID, toolCallID))
	notification := map[string]any{
		"sessionId": sessionID,
		"update": map[string]any{
			"sessionUpdate": "tool_result",
			"toolResult": map[string]any{
				"toolCallId": toolCallID,
				"result":     result,
			},
		},
	}
	return s.writeNotification("session/update", notification)
}

// sendAgentMessageChunk emits a session/update notification with an agent message chunk.
// This streams text content from the agent to the client as it's generated
func (s *acpServer) sendAgentMessageChunk(sessionID, text string) error {
	s.trace(fmt.Sprintf("sendAgentMessageChunk: session=%s, text=%s", sessionID, text))
	notification := map[string]any{
		"sessionId": sessionID,
		"update": map[string]any{
			"sessionUpdate": "agent_message_chunk",
			"content": map[string]any{
				"type": "text",
				"text": text,
			},
		},
	}
	return s.writeNotification("session/update", notification)
}

// nextSessionID generates a unique session ID using a timestamp and sequence number
func (s *acpServer) nextSessionID() string {
	s.sessionIDSeq++
	id := fmt.Sprintf("sess_%d_%d", time.Now().UnixNano(), s.sessionIDSeq)
	s.trace(fmt.Sprintf("nextSessionID: generated %s", id))
	return id
}

// extractUserText extracts and concatenates text content from content blocks
// It filters out non-text blocks and trims whitespace from each text block
// readFileFromURI attempts to read file contents from a file:// URI
func readFileFromURI(uri string) (string, error) {
	parsedURL, err := url.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("invalid URI: %v", err)
	}

	if parsedURL.Scheme != "file" {
		return "", fmt.Errorf("unsupported URI scheme: %s", parsedURL.Scheme)
	}

	// Get the file path from the URI
	filePath := parsedURL.Path

	// Read the file contents
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %v", err)
	}

	return string(content), nil
}

// extractUserText creates a single string from all content blocks
func extractUserText(blocks []contentBlock) string {
	var parts []string
	for _, b := range blocks {
		switch b.Type {
		case "text":
			if strings.TrimSpace(b.Text) != "" {
				parts = append(parts, b.Text)
			}
		case "resource_link":
			// Build resource context information
			resourceInfo := fmt.Sprintf("=== Resource: %s ===\n", b.Name)

			// Add metadata
			if b.Title != "" {
				resourceInfo += fmt.Sprintf("Title: %s\n", b.Title)
			}
			if b.Description != "" {
				resourceInfo += fmt.Sprintf("Description: %s\n", b.Description)
			}
			resourceInfo += fmt.Sprintf("URI: %s\n", b.URI)
			if b.MimeType != "" {
				resourceInfo += fmt.Sprintf("Type: %s\n", b.MimeType)
			}
			if b.Size != nil {
				resourceInfo += fmt.Sprintf("Size: %d bytes\n", *b.Size)
			}

			// Try to read file contents if it's a file:// URI
			if strings.HasPrefix(b.URI, "file://") {
				content, err := readFileFromURI(b.URI)
				if err != nil {
					resourceInfo += fmt.Sprintf("\n[Error reading file: %v]\n", err)
				} else {
					// Limit content size for very large files
					const maxContentSize = 50000 // 50KB limit for inline content
					if len(content) > maxContentSize {
						content = content[:maxContentSize] + "\n\n[... truncated to 50KB ...]"
					}
					resourceInfo += fmt.Sprintf("\n--- File Contents ---\n%s\n--- End of File ---\n", content)
				}
			} else {
				resourceInfo += "\n[External resource - content not available]\n"
			}

			resourceInfo += "=== End Resource ===\n"
			parts = append(parts, resourceInfo)
		}
	}
	result := strings.Join(parts, "\n")
	return result
}
