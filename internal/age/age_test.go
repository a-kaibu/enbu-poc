package age_test

import (
	"testing"

	"github.com/a-kaibu/enbu-poc/internal/age"
)

func TestKeyGenAndEncryptDecrypt(t *testing.T) {
	kp, err := age.GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}

	plaintext := []byte("DATABASE_URL=postgres://secret")

	ciphertext, err := age.EncryptForPublicKeys(plaintext, []string{kp.PublicKey})
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	decrypted, err := age.Decrypt(ciphertext, kp.Identity)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Fatalf("got %q, want %q", decrypted, plaintext)
	}
}

func TestMultipleRecipients(t *testing.T) {
	kp1, _ := age.GenerateKeyPair()
	kp2, _ := age.GenerateKeyPair()

	plaintext := []byte("shared secret")

	ciphertext, err := age.EncryptForPublicKeys(plaintext, []string{kp1.PublicKey, kp2.PublicKey})
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// Both should be able to decrypt
	dec1, err := age.Decrypt(ciphertext, kp1.Identity)
	if err != nil {
		t.Fatalf("Decrypt with kp1: %v", err)
	}
	if string(dec1) != string(plaintext) {
		t.Fatalf("kp1: got %q, want %q", dec1, plaintext)
	}

	dec2, err := age.Decrypt(ciphertext, kp2.Identity)
	if err != nil {
		t.Fatalf("Decrypt with kp2: %v", err)
	}
	if string(dec2) != string(plaintext) {
		t.Fatalf("kp2: got %q, want %q", dec2, plaintext)
	}
}
