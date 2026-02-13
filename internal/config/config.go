package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// defaultConfigDir is the base directory for confluence-mgmt configuration.
const defaultConfigDir = ".config/confluence-mgmt"

// defaultConfigFile is the config file name within the config directory.
const defaultConfigFile = "config.yaml"

// Config holds the global user configuration for confluence-mgmt.
type Config struct {
	ActiveSpace  string `yaml:"active_space"`
	InstanceURL  string `yaml:"instance_url,omitempty"`
	InstanceType string `yaml:"instance_type,omitempty"` // "cloud" or "server"
	AuthType     string `yaml:"auth_type,omitempty"`     // "basic" or "bearer"
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{}
}

// ConfigManager handles reading and writing the config file.
type ConfigManager struct {
	configPath string
}

// NewConfigManager creates a ConfigManager with the default config path (~/.config/confluence-mgmt/config.yaml).
func NewConfigManager() (*ConfigManager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("getting home directory: %w", err)
	}

	configPath := filepath.Join(home, defaultConfigDir, defaultConfigFile)
	return &ConfigManager{configPath: configPath}, nil
}

// NewConfigManagerWithPath creates a ConfigManager with a custom config file path.
func NewConfigManagerWithPath(configPath string) *ConfigManager {
	return &ConfigManager{configPath: configPath}
}

// ConfigPath returns the path to the config file.
func (m *ConfigManager) ConfigPath() string {
	return m.configPath
}

// GetConfig reads the config file and returns the config.
// If the file doesn't exist, returns default config.
func (m *ConfigManager) GetConfig() (Config, error) {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return DefaultConfig(), nil
		}
		return Config{}, fmt.Errorf("reading config file: %w", err)
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config file: %w", err)
	}

	return cfg, nil
}

// Exists returns true if the config file exists.
func (m *ConfigManager) Exists() bool {
	_, err := os.Stat(m.configPath)
	return err == nil
}

// saveConfig writes the config to disk, creating directories as needed.
func (m *ConfigManager) saveConfig(cfg Config) error {
	dir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(m.configPath, data, 0o644); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}

// SetActiveSpace updates the active space key in the config.
func (m *ConfigManager) SetActiveSpace(spaceKey string) error {
	cfg, err := m.GetConfig()
	if err != nil {
		return err
	}

	cfg.ActiveSpace = spaceKey
	return m.saveConfig(cfg)
}

// SetInstanceURL updates the instance URL in the config.
func (m *ConfigManager) SetInstanceURL(instanceURL string) error {
	cfg, err := m.GetConfig()
	if err != nil {
		return err
	}

	cfg.InstanceURL = instanceURL
	return m.saveConfig(cfg)
}

// SetInstanceType updates the instance type in the config.
func (m *ConfigManager) SetInstanceType(instanceType string) error {
	cfg, err := m.GetConfig()
	if err != nil {
		return err
	}

	cfg.InstanceType = instanceType
	return m.saveConfig(cfg)
}

// SetAuthType updates the auth type in the config.
func (m *ConfigManager) SetAuthType(authType string) error {
	cfg, err := m.GetConfig()
	if err != nil {
		return err
	}

	cfg.AuthType = authType
	return m.saveConfig(cfg)
}
