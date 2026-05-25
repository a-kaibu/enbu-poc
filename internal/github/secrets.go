package github

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/crypto/nacl/box"
)

type PublicKey struct {
	KeyID string `json:"key_id"`
	Key   string `json:"key"`
}

func (c *Client) GetRepoPublicKey(ctx context.Context, owner, repo string) (*PublicKey, error) {
	path := fmt.Sprintf("/repos/%s/%s/actions/secrets/public-key", owner, repo)
	resp, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("getting public key: status %d: %s", resp.StatusCode, body)
	}

	var pk PublicKey
	if err := json.NewDecoder(resp.Body).Decode(&pk); err != nil {
		return nil, err
	}
	return &pk, nil
}

func (c *Client) SetSecret(ctx context.Context, owner, repo, name, value string) error {
	pk, err := c.GetRepoPublicKey(ctx, owner, repo)
	if err != nil {
		return err
	}

	encrypted, err := encryptSecret(pk.Key, value)
	if err != nil {
		return fmt.Errorf("encrypting secret: %w", err)
	}

	payload := map[string]string{
		"encrypted_value": encrypted,
		"key_id":          pk.KeyID,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/repos/%s/%s/actions/secrets/%s", owner, repo, name)
	resp, err := c.do(ctx, http.MethodPut, path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("setting secret: status %d: %s", resp.StatusCode, respBody)
	}

	return nil
}

func (c *Client) GetSecret(ctx context.Context, owner, repo, name string) (string, error) {
	path := fmt.Sprintf("/repos/%s/%s/actions/secrets/%s", owner, repo, name)
	resp, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", nil
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("getting secret: status %d", resp.StatusCode)
	}

	var result struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.Name, nil
}

func (c *Client) DispatchWorkflow(ctx context.Context, owner, repo, eventType string, payload map[string]string) error {
	body, err := json.Marshal(map[string]any{
		"event_type":     eventType,
		"client_payload": payload,
	})
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/repos/%s/%s/dispatches", owner, repo)
	resp, err := c.do(ctx, http.MethodPost, path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("dispatching workflow: status %d: %s", resp.StatusCode, respBody)
	}

	return nil
}

func encryptSecret(publicKeyB64, secret string) (string, error) {
	publicKeyBytes, err := base64.StdEncoding.DecodeString(publicKeyB64)
	if err != nil {
		return "", fmt.Errorf("decoding public key: %w", err)
	}

	var publicKey [32]byte
	copy(publicKey[:], publicKeyBytes)

	encrypted, err := box.SealAnonymous(nil, []byte(secret), &publicKey, rand.Reader)
	if err != nil {
		return "", fmt.Errorf("sealing: %w", err)
	}

	return base64.StdEncoding.EncodeToString(encrypted), nil
}
