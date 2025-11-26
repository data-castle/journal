package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the global journal configuration
type Config struct {
	DefaultJournal string              `yaml:"default_journal"`
	Journals       map[string]*Journal `yaml:"journals"`
}

// Journal represents a single journal configuration
type Journal struct {
	Name string `yaml:"name"`
	Path string `yaml:"path"`
}

// GetConfigPathFunc is the function used to get the config path
// It's exported so tests can override it
var GetConfigPathFunc = getConfigPathDefault

// GetConfigPath returns the path to the config file
func GetConfigPath() (string, error) {
	return GetConfigPathFunc()
}

// getConfigPathDefault is the default implementation
func getConfigPathDefault() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".journal", "config.yaml"), nil
}

// NewConfig creates a new empty configuration
func NewConfig() *Config {
	return &Config{
		Journals: make(map[string]*Journal),
	}
}

// LoadConfig loads the configuration file
func LoadConfig() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &Config{
			Journals: make(map[string]*Journal),
		}, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("config file is empty (possibly corrupted)")
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if config.Journals == nil {
		return nil, fmt.Errorf("config file is corrupted: 'journals' field is null")
	}

	return &config, nil
}

// Save saves the configuration file
func (c *Config) Save() error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// AddJournal adds a new journal to the configuration
func (c *Config) AddJournal(journal *Journal) error {
	if journal.Name == "" {
		return fmt.Errorf("journal name is required")
	}
	if _, exists := c.Journals[journal.Name]; exists {
		return fmt.Errorf("journal %s already exists", journal.Name)
	}

	c.Journals[journal.Name] = journal
	if len(c.Journals) == 1 {
		c.DefaultJournal = journal.Name
	}

	return nil
}

// GetJournal returns a journal by name
func (c *Config) GetJournal(name string) (*Journal, error) {
	if len(c.Journals) == 0 {
		return nil, fmt.Errorf("no journals configured")
	}
	journal, exists := c.Journals[name]
	if !exists {
		return nil, fmt.Errorf("journal %s not found", name)
	}
	return journal, nil
}

// GetDefaultJournal returns the default journal
func (c *Config) GetDefaultJournal() (*Journal, error) {
	if len(c.Journals) == 0 {
		return nil, fmt.Errorf("no journals configured")
	}
	if c.DefaultJournal == "" {
		return nil, fmt.Errorf("no default journal set")
	}

	return c.GetJournal(c.DefaultJournal)
}

// SetDefaultJournal sets the default journal
func (c *Config) SetDefaultJournal(name string) error {
	if len(c.Journals) == 0 {
		return fmt.Errorf("no journals configured")
	}
	if _, exists := c.Journals[name]; !exists {
		return fmt.Errorf("journal %s not found", name)
	}
	c.DefaultJournal = name
	return nil
}

// RemoveJournal removes a journal from the configuration
func (c *Config) RemoveJournal(name string) error {
	if len(c.Journals) == 0 {
		return fmt.Errorf("no journals configured")
	}
	if _, exists := c.Journals[name]; !exists {
		return fmt.Errorf("journal %s not found", name)
	}

	// Prevent deletion of the default journal
	if c.DefaultJournal == name {
		return fmt.Errorf("cannot remove default journal %s; use set-default to change default first", name)
	}

	delete(c.Journals, name)

	return nil
}

// ListJournals returns all journal names
func (c *Config) ListJournals() []string {
	var names []string
	for name := range c.Journals {
		names = append(names, name)
	}
	return names
}
