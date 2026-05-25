package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user" {
			t.Errorf("path = %q, want /user", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer test-token")
		}
		if got := r.Header.Get("X-GitHub-Api-Version"); got != "2022-11-28" {
			t.Errorf("API version = %q, want 2022-11-28", got)
		}
		json.NewEncoder(w).Encode(User{Login: "testuser"})
	}))
	defer server.Close()

	client := &Client{
		token:      "test-token",
		httpClient: http.DefaultClient,
	}
	origBase := apiBaseURL
	defer func() { apiBaseURL = origBase }()
	apiBaseURL = server.URL

	user, err := client.GetUser(context.Background())
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if user.Login != "testuser" {
		t.Errorf("Login = %q, want %q", user.Login, "testuser")
	}
}

func TestGetRepoPublicKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/myorg/myrepo/actions/secrets/public-key" {
			t.Errorf("path = %q", r.URL.Path)
		}
		json.NewEncoder(w).Encode(PublicKey{
			KeyID: "key-123",
			Key:   "dGVzdGtleQ==",
		})
	}))
	defer server.Close()

	client := &Client{token: "tok", httpClient: http.DefaultClient}
	origBase := apiBaseURL
	defer func() { apiBaseURL = origBase }()
	apiBaseURL = server.URL

	pk, err := client.GetRepoPublicKey(context.Background(), "myorg", "myrepo")
	if err != nil {
		t.Fatalf("GetRepoPublicKey: %v", err)
	}
	if pk.KeyID != "key-123" {
		t.Errorf("KeyID = %q, want %q", pk.KeyID, "key-123")
	}
}

func TestDispatchWorkflow(t *testing.T) {
	var receivedPayload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/myorg/myrepo/dispatches" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		json.NewDecoder(r.Body).Decode(&receivedPayload)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := &Client{token: "tok", httpClient: http.DefaultClient}
	origBase := apiBaseURL
	defer func() { apiBaseURL = origBase }()
	apiBaseURL = server.URL

	err := client.DispatchWorkflow(context.Background(), "myorg", "myrepo", "test-event", nil)
	if err != nil {
		t.Fatalf("DispatchWorkflow: %v", err)
	}
	if receivedPayload["event_type"] != "test-event" {
		t.Errorf("event_type = %v, want test-event", receivedPayload["event_type"])
	}
	if receivedPayload["client_payload"] == nil {
		t.Error("client_payload should not be nil")
	}
}
