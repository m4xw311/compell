package config

import (
	"os"
	"path/filepath"

	"github.com/m4xw311/compell/errors"
	"gopkg.in/yaml.v3"
)

type FilesystemAccess struct {
	Hidden   []string `yaml:"hidden"`
	ReadOnly []string `yaml:"read_only"`
}

type MCPServer struct {
	Name    string   `yaml:"name"`
	Command string   `yaml:"command"`
	Args    []string `yaml:"args"`
}

type Toolset struct {
	Name  string   `yaml:"name"`
	Tools []string `yaml:"tools"`
}

type Config struct {
	LLMClient            string           `yaml:"llm"`
	Model                string           `yaml:"model"`
	Toolsets             []Toolset        `yaml:"toolsets"`
	AdditionalMCPServers []MCPServer      `yaml:"additional_mcp_servers"`
	AllowedCommands      []string         `yaml:"allowed_commands"`
	FilesystemAccess     FilesystemAccess `yaml:"filesystem_access"`
}

// LoadConfig loads configuration from the user's home directory and the current
// working directory, with the latter taking precedence.
func LoadConfig() (*Config, error) {
	cfg := &Config{}

	// Default .compell directory to be hidden
	cfg.FilesystemAccess.Hidden = append(cfg.FilesystemAccess.Hidden, ".compell", ".compell/**")

	// Load user-level config first
	home, err := os.UserHomeDir()
	if err == nil {
		userConfigPath := filepath.Join(home, ".compell", "config.yaml")
		if _, err := os.Stat(userConfigPath); err == nil {
			if err := loadFromFile(userConfigPath, cfg); err != nil {
				return nil, errors.Wrapf(err, "error loading user config")
			}
		}
	}

	// Load project-level config, overriding user-level
	wd, err := os.Getwd()
	if err != nil {
		return nil, errors.Wrapf(err, "could not get working directory")
	}
	projectConfigPath := filepath.Join(wd, ".compell", "config.yaml")
	if _, err := os.Stat(projectConfigPath); err == nil {
		if err := loadFromFile(projectConfigPath, cfg); err != nil {
			return nil, errors.Wrapf(err, "error loading project config")
		}
	}

	return cfg, nil
}

func loadFromFile(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	// Note: Unmarshal will overwrite fields present in the YAML. This provides
	// a simple merge where project-level config replaces user-level.
	// A more sophisticated merge could be implemented if needed.
	return yaml.Unmarshal(data, cfg)
}

// GetToolset finds a toolset by name. Returns the "default" toolset if the
// named one is not found or if an empty name is provided.
func (c *Config) GetToolset(name string) (*Toolset, error) {
	if name == "" {
		name = "default"
	}
	for _, ts := range c.Toolsets {
		if ts.Name == name {
			return &ts, nil
		}
	}
	if name == "default" {
		return nil, errors.New("mandatory 'default' toolset not found in configuration")
	}
	// Fallback to default if a specific toolset was requested but not found
	return c.GetToolset("default")
}
