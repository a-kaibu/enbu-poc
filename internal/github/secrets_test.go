package github

import (
	"crypto/rand"
	"encoding/base64"
	"testing"

	"golang.org/x/crypto/nacl/box"
)

func TestEncryptSecret(t *testing.T) {
	pub, priv, err := box.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}

	pubB64 := base64.StdEncoding.EncodeToString(pub[:])

	encrypted, err := encryptSecret(pubB64, "my-secret-value")
	if err != nil {
		t.Fatalf("encryptSecret: %v", err)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		t.Fatalf("decoding encrypted: %v", err)
	}

	plaintext, ok := box.OpenAnonymous(nil, ciphertext, pub, priv)
	if !ok {
		t.Fatal("failed to decrypt")
	}

	if string(plaintext) != "my-secret-value" {
		t.Errorf("got %q, want %q", plaintext, "my-secret-value")
	}
}

func TestEncryptSecretInvalidKey(t *testing.T) {
	_, err := encryptSecret("not-valid-base64!!!", "secret")
	if err == nil {
		t.Fatal("expected error for invalid key")
	}
}
