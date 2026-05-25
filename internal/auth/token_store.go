package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/a-kaibu/enbu-poc/internal/config"
)

type StoredToken struct {
	AccessToken string `json:"access_token"`
	Username    string `json:"username"`
}

func SaveToken(token *StoredToken) error {
	dir := config.DataDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating data directory: %w", err)
	}

	data, err := json.Marshal(token)
	if err != nil {
		return err
	}

	path := filepath.Join(dir, "token.json")
	return os.WriteFile(path, data, 0o600)
}

func LoadToken() (*StoredToken, error) {
	path := filepath.Join(config.DataDir(), "token.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading token: %w (run 'enbu auth' first)", err)
	}

	var token StoredToken
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("parsing token: %w", err)
	}

	return &token, nil
}
