package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/m4xw311/compell/errors"
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
		return nil, errors.Wrapf(err, "could not read session file %s", path)
	}

	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, errors.Wrapf(err, "could not parse session file %s", path)
	}
	s.path = path
	return &s, nil
}

// Save writes the current session state to disk.
func (s *Session) Save() error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return errors.Wrapf(err, "failed to serialize session")
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
		return "", errors.Wrapf(err, "could not create session directory")
	}
	return filepath.Join(sessionDir, fmt.Sprintf("%s.json", name)), nil
}
