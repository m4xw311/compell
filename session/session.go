package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Message struct {
	Role    string `json:"role"` // "user", "assistant", "tool"
	Content string `json:"content"`
	// For future tool call implementation
	// ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

type Session struct {
	Name     string    `json:"name"`
	Messages []Message `json:"messages"`
	path     string
}

// New creates a new session.
func New(name string) (*Session, error) {
	path, err := getSessionPath(name)
	if err != nil {
		return nil, err
	}
	return &Session{
		Name:     name,
		Messages: []Message{},
		path:     path,
	}, nil
}

// Load loads an existing session from disk.
func Load(name string) (*Session, error) {
	path, err := getSessionPath(name)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read session file %s: %w", path, err)
	}

	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("could not parse session file %s: %w", path, err)
	}
	s.path = path
	return &s, nil
}

// Save writes the current session state to disk.
func (s *Session) Save() error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize session: %w", err)
	}
	return os.WriteFile(s.path, data, 0644)
}

// AddMessage appends a message to the session history.
func (s *Session) AddMessage(msg Message) {
	s.Messages = append(s.Messages, msg)
}

func getSessionPath(name string) (string, error) {
	sessionDir := filepath.Join(".compell", "sessions")
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return "", fmt.Errorf("could not create session directory: %w", err)
	}
	return filepath.Join(sessionDir, fmt.Sprintf("%s.json", name)), nil
}
