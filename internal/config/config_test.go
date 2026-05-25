package config

import (
	"testing"
)

func TestParseGitRemoteSSH(t *testing.T) {
	owner, repo, err := parseGitRemote("git@github.com:a-kaibu/enbu-poc.git")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if owner != "a-kaibu" || repo != "enbu-poc" {
		t.Fatalf("got %s/%s, want a-kaibu/enbu-poc", owner, repo)
	}
}

func TestParseGitRemoteHTTPS(t *testing.T) {
	owner, repo, err := parseGitRemote("https://github.com/a-kaibu/enbu-poc.git")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if owner != "a-kaibu" || repo != "enbu-poc" {
		t.Fatalf("got %s/%s, want a-kaibu/enbu-poc", owner, repo)
	}
}

func TestParseGitRemoteHTTPSNoSuffix(t *testing.T) {
	owner, repo, err := parseGitRemote("https://github.com/a-kaibu/enbu-poc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if owner != "a-kaibu" || repo != "enbu-poc" {
		t.Fatalf("got %s/%s, want a-kaibu/enbu-poc", owner, repo)
	}
}
