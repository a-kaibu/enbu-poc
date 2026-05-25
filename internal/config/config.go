package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type Config struct {
	Owner string
	Repo  string
}

func Load() (*Config, error) {
	owner, repo, err := detectRepo()
	if err != nil {
		return nil, fmt.Errorf("detecting repository: %w", err)
	}
	return &Config{Owner: owner, Repo: repo}, nil
}

func detectRepo() (string, string, error) {
	out, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return "", "", fmt.Errorf("git remote not found: %w", err)
	}
	return parseGitRemote(strings.TrimSpace(string(out)))
}

func parseGitRemote(url string) (string, string, error) {
	// Handle SSH: git@github.com:owner/repo.git
	if strings.HasPrefix(url, "git@") {
		parts := strings.SplitN(url, ":", 2)
		if len(parts) != 2 {
			return "", "", fmt.Errorf("invalid SSH remote: %s", url)
		}
		return splitOwnerRepo(strings.TrimSuffix(parts[1], ".git"))
	}

	// Handle HTTPS: https://github.com/owner/repo.git
	url = strings.TrimSuffix(url, ".git")
	parts := strings.Split(url, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid remote URL: %s", url)
	}
	return parts[len(parts)-2], parts[len(parts)-1], nil
}

func splitOwnerRepo(path string) (string, string, error) {
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid owner/repo: %s", path)
	}
	return parts[0], parts[1], nil
}

func DataDir() string {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "enbu")
	}
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "enbu")
	case "windows":
		return filepath.Join(os.Getenv("LOCALAPPDATA"), "enbu")
	default:
		return filepath.Join(os.Getenv("HOME"), ".local", "share", "enbu")
	}
}
