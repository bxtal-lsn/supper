package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/bxtal-lsn/supper/internal/age"
)

// Config represents the application configuration
type Config struct {
	KeyPath            string        `json:"key_path"`
	EncryptedKeyPath   string        `json:"encrypted_key_path"`
	AutoDeleteInterval time.Duration `json:"auto_delete_interval"`
	EditorCommand      string        `json:"editor_command"`
	DefaultRecipients  string        `json:"default_recipients"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		KeyPath:            age.DefaultKeyPath(),
		EncryptedKeyPath:   age.DefaultEncryptedKeyPath(),
		AutoDeleteInterval: 30 * time.Minute,
		EditorCommand:      "default", // Uses EDITOR environment variable if available
		DefaultRecipients:  "",
	}
}

// ConfigPath returns the path to the configuration file
func ConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config directory: %w", err)
	}

	return filepath.Join(configDir, "sops-tui", "config.json"), nil
}

// Load loads the configuration from disk
func Load() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	// If the config file doesn't exist, return the default config
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return DefaultConfig(), nil
	}

	// Read the config file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse the JSON
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// Save saves the configuration to disk
func Save(config *Config) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}

	// Create the config directory if it doesn't exist
	configDir := filepath.Dir(path)
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal the config to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write the config file
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
