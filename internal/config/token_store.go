package config

import (
	"fmt"
	"os"

	"golang.org/x/oauth2"
	"gopkg.in/yaml.v3"
)

type tokenRecord struct {
	AccessToken  string `yaml:"access_token"`
	RefreshToken string `yaml:"refresh_token,omitempty"`
	TokenType    string `yaml:"token_type,omitempty"`
	Expiry       string `yaml:"expiry,omitempty"`
}

func LoadToken() (*oauth2.Token, error) {
	path, err := ResolveTokenFile()
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read token file: %w", err)
	}
	rec := &tokenRecord{}
	if err := yaml.Unmarshal(raw, rec); err != nil {
		return nil, fmt.Errorf("parse token file: %w", err)
	}
	if rec.AccessToken == "" {
		return nil, fmt.Errorf("token file is empty")
	}
	return &oauth2.Token{
		AccessToken:  rec.AccessToken,
		RefreshToken: rec.RefreshToken,
		TokenType:    rec.TokenType,
	}, nil
}

func SaveToken(token *oauth2.Token) error {
	if token == nil {
		return fmt.Errorf("token is nil")
	}
	if _, err := EnsureConfigDir(); err != nil {
		return err
	}
	path, err := ResolveTokenFile()
	if err != nil {
		return err
	}
	rec := &tokenRecord{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
	}
	raw, err := yaml.Marshal(rec)
	if err != nil {
		return fmt.Errorf("marshal token: %w", err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return fmt.Errorf("write token file: %w", err)
	}
	return nil
}
