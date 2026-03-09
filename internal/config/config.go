package config

import (
	"errors"
	"fmt"
	"net/url"
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
	BaseURL    string `yaml:"base_url"`
	Repository string `yaml:"repository,omitempty"`
	Owner      string `yaml:"owner,omitempty"`
	Repo       string `yaml:"repo,omitempty"`
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

	repoSpec := strings.TrimSpace(cfg.Repository)
	if repoSpec == "" && strings.TrimSpace(cfg.Owner) != "" && strings.TrimSpace(cfg.Repo) != "" {
		repoSpec = strings.TrimSpace(cfg.Owner) + "/" + strings.TrimSpace(cfg.Repo)
	}

	owner, repo, err := ParseRepositorySpec(repoSpec)
	if err != nil {
		return nil, err
	}
	cfg.Owner = owner
	cfg.Repo = repo
	cfg.Repository = owner + "/" + repo

	return cfg, nil
}

func SaveExample(path string) error {
	example := Config{
		BaseURL:    "https://api.github.com",
		Repository: "your_owner/your_repo",
	}
	return Save(path, &example)
}

func ParseRepositorySpec(spec string) (string, string, error) {
	value := strings.TrimSpace(spec)
	if value == "" {
		return "", "", errors.New("missing repository in config file")
	}

	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		parsed, err := url.Parse(value)
		if err != nil {
			return "", "", fmt.Errorf("invalid repository url: %w", err)
		}
		if !strings.EqualFold(parsed.Host, "github.com") {
			return "", "", errors.New("repository url must use github.com")
		}
		parts := splitPath(parsed.Path)
		if len(parts) < 2 {
			return "", "", errors.New("repository url must include owner/repo")
		}
		return parts[0], trimRepoSuffix(parts[1]), nil
	}

	parts := splitPath(value)
	if len(parts) != 2 {
		return "", "", errors.New("repository must be owner/repo or a github repository url")
	}
	return parts[0], trimRepoSuffix(parts[1]), nil
}

func splitPath(value string) []string {
	clean := strings.TrimSpace(value)
	clean = strings.Trim(clean, "/")
	if clean == "" {
		return nil
	}
	parts := strings.Split(clean, "/")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
	}
	return result
}

func trimRepoSuffix(value string) string {
	repo := strings.TrimSpace(value)
	repo = strings.TrimSuffix(repo, ".git")
	return repo
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
	if strings.TrimSpace(cfg.Repository) == "" && strings.TrimSpace(cfg.Owner) != "" && strings.TrimSpace(cfg.Repo) != "" {
		cfg.Repository = strings.TrimSpace(cfg.Owner) + "/" + strings.TrimSpace(cfg.Repo)
	}
	if strings.TrimSpace(cfg.Repository) == "" {
		return errors.New("config repository is empty")
	}
	owner, repo, err := ParseRepositorySpec(cfg.Repository)
	if err != nil {
		return err
	}
	cfg.Repository = owner + "/" + repo
	cfg.Owner = ""
	cfg.Repo = ""
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
