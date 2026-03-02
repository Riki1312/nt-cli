package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Token holds OAuth credentials for the Notion MCP server.
type Token struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	TokenType    string    `json:"token_type"`
	Expiry       time.Time `json:"expiry,omitempty"`
}

// IsExpired returns true if the token has expired (with a 30-second buffer).
func (t *Token) IsExpired() bool {
	if t.Expiry.IsZero() {
		return false
	}
	return time.Now().After(t.Expiry.Add(-30 * time.Second))
}

// ClientRegistration holds the dynamically registered OAuth client info.
type ClientRegistration struct {
	ClientID string `json:"client_id"`
}

var ErrNoToken = errors.New("not logged in; run 'nt login'")

func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("finding home directory: %w", err)
	}
	dir := filepath.Join(home, ".config", "nt")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("creating config directory: %w", err)
	}
	return dir, nil
}

func LoadToken() (*Token, error) {
	dir, err := configDir()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(dir, "token.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNoToken
		}
		return nil, fmt.Errorf("reading token file: %w", err)
	}
	var tok Token
	if err := json.Unmarshal(data, &tok); err != nil {
		return nil, fmt.Errorf("parsing token file: %w", err)
	}
	return &tok, nil
}

func SaveToken(tok *Token) error {
	dir, err := configDir()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(tok, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling token: %w", err)
	}
	path := filepath.Join(dir, "token.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing token file: %w", err)
	}
	return nil
}

func LoadClientRegistration() (*ClientRegistration, error) {
	dir, err := configDir()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(dir, "client.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading client file: %w", err)
	}
	var reg ClientRegistration
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("parsing client file: %w", err)
	}
	return &reg, nil
}

func SaveClientRegistration(reg *ClientRegistration) error {
	dir, err := configDir()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling client registration: %w", err)
	}
	path := filepath.Join(dir, "client.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing client file: %w", err)
	}
	return nil
}

func DeleteToken() error {
	dir, err := configDir()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, "token.json")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing token file: %w", err)
	}
	return nil
}
