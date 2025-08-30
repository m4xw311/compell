package acp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/m4xw311/compell/agent"
	"github.com/m4xw311/compell/session"
	"github.com/m4xw311/compell/tools"
)

// Run starts the Agent Client Protocol server over stdio using JSON-RPC
// It implements a minimal subset of ACP:
// - initialize
// - session/new
// - session/prompt (emits session/update notifications with agent_message_chunk, tool_call, and tool_result)
// Notes:
// - This implementation intentionally avoids writing anything to stdout except JSON-RPC messages.
// - Any debug or informational logs should go to trace file if needed.
func Run(ctx context.Context, compellAgent *agent.Agent, traceFlag *bool) error {
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
		sessions:     make(map[string][]session.Message),
		sessionIDSeq: 0,
		stdinReader:  bufio.NewReader(os.Stdin),
		stdoutWriter: bufio.NewWriter(os.Stdout),
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

type jsonrpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id,omitempty"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type jsonrpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonrpcError   `json:"error,omitempty"`
}

type jsonrpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// ---- acpServer ----

type acpServer struct {
	ctx          context.Context
	agent        *agent.Agent
	sessions     map[string][]session.Message
	sessionsLock sync.Mutex
	sessionIDSeq int64

	stdinReader  *bufio.Reader
	stdoutWriter *bufio.Writer
	writeLock    *sync.Mutex
	trace        func(string)
}

// readFramedMessage reads a single JSON-RPC payload
func (s *acpServer) readFramedMessage() ([]byte, error) {
	s.trace("readFramedMessage: starting")
	// JSON-RPC requests and responses are newline-delimited JSONs.
	line, _, err := s.stdinReader.ReadLine()
	if err != nil {
		s.trace(fmt.Sprintf("readFramedMessage: error reading message: %v", err))
		return nil, err
	}

	s.trace(fmt.Sprintf("readFramedMessage: successfully read direct JSON message of length %d: %s", len(line), string(line)))
	return line, nil
}

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
	if _, err := s.stdoutWriter.Write(data); err != nil {
		s.trace(fmt.Sprintf("writeFramedJSON: write error: %v", err))
		return err
	}
	// JSON-RPC requests and responses are newline-delimited JSONs.
	// Write newline to stdout to inform client that message is complete
	if _, err := s.stdoutWriter.WriteString("\n"); err != nil {
		s.trace(fmt.Sprintf("writeFramedJSON: write error: %v", err))
		return err
	}
	err = s.stdoutWriter.Flush()
	if err != nil {
		s.trace(fmt.Sprintf("writeFramedJSON: flush error: %v", err))
		return err
	}
	s.trace("writeFramedJSON: successfully wrote message")
	return nil
}

func (s *acpServer) writeResponseOK(id any, result json.RawMessage) error {
	s.trace("writeResponseOK: starting")
	resp := jsonrpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	return s.writeFramedJSON(resp)
}

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

func (s *acpServer) handleInitialize(req *jsonrpcRequest) {
	s.trace("handleInitialize: starting")
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
			"loadSession": false, // TODO: Implement session loading in agent
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

func (s *acpServer) handleSessionNew(req *jsonrpcRequest) {
	s.trace("handleSessionNew: starting")
	// params: { cwd: string, mcpServers: [] }
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

	// Create a new session ID and in-memory history
	sid := s.nextSessionID()
	s.trace(fmt.Sprintf("handleSessionNew: created session ID: %s", sid))

	s.sessionsLock.Lock()
	s.sessions[sid] = []session.Message{}
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

// contentBlock represents a content block in ACP prompt requests.
// For this minimal implementation, we only handle text blocks.
type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	// We ignore other fields for minimal MVP
}

func (s *acpServer) handleSessionPrompt(req *jsonrpcRequest) {
	s.trace("handleSessionPrompt: starting")
	// params: { sessionId: string, prompt: []ContentBlock }
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
	msgs, ok := s.sessions[p.SessionID]
	s.sessionsLock.Unlock()
	if !ok {
		s.trace("handleSessionPrompt: unknown sessionId")
		_ = s.writeResponseError(req.ID, -32602, "Invalid params", "unknown sessionId")
		return
	}

	// Extract user text from prompt content blocks (text only for MVP)
	s.trace(fmt.Sprintf("handleSessionPrompt: extracting user text from prompt: %+v", p.Prompt))
	userText := extractUserText(p.Prompt)
	s.trace(fmt.Sprintf("handleSessionPrompt: extracted user text: %s", userText))

	// Append user message
	s.trace("handleSessionPrompt: appending user message")
	userMsg := session.Message{Role: "user", Content: userText}
	msgs = append(msgs, userMsg)

	// Main loop: LLM -> Tool -> LLM ... (similar to agent.go's processTurn)
	for {
		// Call LLM client with current history and available tools
		s.trace("handleSessionPrompt: calling LLM client with messages")
		reply, err := s.agent.LLMClient.Chat(s.ctx, msgs, s.agent.AvailableTools)
		if err != nil {
			s.trace(fmt.Sprintf("handleSessionPrompt: LLM chat failed: %v", err))
			_ = s.writeResponseError(req.ID, -32603, "Internal error", fmt.Sprintf("LLM chat failed: %v", err))
			return
		}
		s.trace(fmt.Sprintf("handleSessionPrompt: LLM client response: %+v", reply))

		// Update history with assistant's response
		s.trace("handleSessionPrompt: updating history with assistant response")
		msgs = append(msgs, *reply)

		// Stream agent message if there's content
		if strings.TrimSpace(reply.Content) != "" {
			s.trace(fmt.Sprintf("handleSessionPrompt: sending agent message chunk with content: %s", reply.Content))
			_ = s.sendAgentMessageChunk(p.SessionID, reply.Content)
		}

		// Check if there are tool calls to execute
		if len(reply.ToolCalls) == 0 {
			s.trace("handleSessionPrompt: no tool calls, ending turn")
			// No tool calls, we're done - save session and exit loop
			s.sessionsLock.Lock()
			s.sessions[p.SessionID] = msgs
			s.sessionsLock.Unlock()
			break
		}

		// Execute tool calls
		s.trace(fmt.Sprintf("handleSessionPrompt: executing %d tool calls", len(reply.ToolCalls)))

		for _, toolCall := range reply.ToolCalls {
			s.trace(fmt.Sprintf("handleSessionPrompt: executing tool call: %s with args: %v", toolCall.Name, toolCall.Args))

			// Send tool_call notification
			_ = s.sendToolCallNotification(p.SessionID, toolCall)

			// Execute the tool
			toolResult, err := s.executeToolCall(toolCall)
			if err != nil {
				s.trace(fmt.Sprintf("handleSessionPrompt: tool execution error for %s: %v", toolCall.Name, err))
				toolResult = fmt.Sprintf("Error executing tool %s: %v", toolCall.Name, err)
			}

			// Send tool_result notification
			_ = s.sendToolResultNotification(p.SessionID, toolCall.ToolCallID, toolResult)

			// Add tool result to messages
			toolMsg := session.Message{
				Role:    "tool",
				Content: toolResult,
				ToolCalls: []session.ToolCall{
					{ToolCallID: toolCall.ToolCallID, Name: toolCall.Name},
				},
			}
			msgs = append(msgs, toolMsg)
		}

		// Continue loop to send tool results back to LLM
		s.trace("handleSessionPrompt: continuing loop after tool execution")
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

// executeToolCall executes a tool and returns its result
func (s *acpServer) executeToolCall(toolCall session.ToolCall) (string, error) {
	s.trace(fmt.Sprintf("executeToolCall: looking for tool %s", toolCall.Name))

	var targetTool tools.Tool
	for _, t := range s.agent.AvailableTools {
		if t.Name() == toolCall.Name {
			targetTool = t
			break
		}
	}

	if targetTool == nil {
		return "", fmt.Errorf("tool '%s' not found in the available toolset", toolCall.Name)
	}

	s.trace(fmt.Sprintf("executeToolCall: executing tool %s with args: %v", toolCall.Name, toolCall.Args))

	// Execute the tool
	result, err := targetTool.Execute(s.ctx, toolCall.Args)
	if err != nil {
		return "", err
	}

	return result, nil
}

// sendToolCallNotification emits a session/update notification for a tool call
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

func (s *acpServer) nextSessionID() string {
	s.sessionIDSeq++
	id := fmt.Sprintf("sess_%d_%d", time.Now().UnixNano(), s.sessionIDSeq)
	s.trace(fmt.Sprintf("nextSessionID: generated %s", id))
	return id
}

func extractUserText(blocks []contentBlock) string {
	var parts []string
	for _, b := range blocks {
		if b.Type == "text" && strings.TrimSpace(b.Text) != "" {
			parts = append(parts, b.Text)
		}
	}
	result := strings.Join(parts, "\n")
	return result
}
