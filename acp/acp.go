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
)

// Run starts the Agent Client Protocol server over stdio using JSON-RPC
// It implements a minimal subset of ACP:
// - initialize
// - session/new
// - session/prompt (emits session/update notifications with agent_message_chunk)
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
		s.trace(fmt.Sprintf("handleInitialize: err : %v", err))
	}
	err = json.Unmarshal(b, &p)
	if err != nil {
		s.trace(fmt.Sprintf("handleInitialize: err : %v", err))
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

	// Call LLM client with current history and available tools
	s.trace(fmt.Sprintf("handleSessionPrompt: calling LLM client with messages: %+v and tools: %+v", msgs, s.agent.AvailableTools))
	reply, err := s.agent.LLMClient.Chat(s.ctx, msgs, s.agent.AvailableTools)
	if err != nil {
		s.trace(fmt.Sprintf("handleSessionPrompt: LLM chat failed: %v", err))
		_ = s.writeResponseError(req.ID, -32603, "Internal error", fmt.Sprintf("LLM chat failed: %v", err))
		return
	}
	s.trace(fmt.Sprintf("handleSessionPrompt: LLM client response: %+v", reply))

	// Update history
	s.trace("handleSessionPrompt: updating history")
	msgs = append(msgs, *reply)

	s.sessionsLock.Lock()
	s.sessions[p.SessionID] = msgs
	s.sessionsLock.Unlock()

	// Stream a single agent_message_chunk notification with the full text
	if strings.TrimSpace(reply.Content) != "" {
		s.trace(fmt.Sprintf("handleSessionPrompt: sending agent message chunk with content: %s", reply.Content))
		_ = s.sendAgentMessageChunk(p.SessionID, reply.Content)
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
