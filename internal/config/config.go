package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	defaultConfigDir  = ".config/ghnote"
	defaultConfigFile = "config.yaml"
	tokenFileName     = "token.yaml"
)

// Config keeps runtime settings for ghnote.
type Config struct {
	ClientID string `yaml:"client_id,omitempty"`
	BaseURL  string `yaml:"base_url"`
	Owner    string `yaml:"owner"`
	Repo     string `yaml:"repo"`
}

func ResolveConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, defaultConfigDir), nil
}

func ResolveConfigFile() (string, error) {
	dir, err := ResolveConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, defaultConfigFile), nil
}

func ResolveTokenFile() (string, error) {
	dir, err := ResolveConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, tokenFileName), nil
}

func EnsureConfigDir() (string, error) {
	dir, err := ResolveConfigDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create config dir: %w", err)
	}
	return dir, nil
}

func ensureParentDir(path string) error {
	dir := filepath.Dir(path)
	if strings.TrimSpace(dir) == "" || dir == "." {
		return nil
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	return nil
}

func Load(configPath string) (*Config, error) {
	raw, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}
	cfg := &Config{}
	if err := yaml.Unmarshal(raw, cfg); err != nil {
		return nil, fmt.Errorf("parse config file: %w", err)
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.github.com"
	}
	if cfg.ClientID == "" {
		cfg.ClientID = os.Getenv("GHNOTE_GITHUB_CLIENT_ID")
	}
	if cfg.Owner == "" || cfg.Repo == "" {
		return nil, errors.New("missing owner/repo in config file")
	}
	return cfg, nil
}

func SaveExample(path string) error {
	example := Config{
		BaseURL: "https://api.github.com",
		Owner:   "your_owner",
		Repo:    "your_repo",
	}
	return Save(path, &example)
}

func Save(path string, cfg *Config) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("config path is empty")
	}
	if cfg == nil {
		return errors.New("config is nil")
	}
	if strings.TrimSpace(cfg.BaseURL) == "" {
		cfg.BaseURL = "https://api.github.com"
	}
	if err := ensureParentDir(path); err != nil {
		return err
	}
	content, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}
	return nil
}
