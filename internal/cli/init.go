package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/a-kaibu/enbu-poc/internal/age"
	"github.com/a-kaibu/enbu-poc/internal/auth"
	"github.com/a-kaibu/enbu-poc/internal/config"
	"github.com/a-kaibu/enbu-poc/internal/oci"
	"github.com/a-kaibu/enbu-poc/internal/tokenlock"
	"github.com/urfave/cli/v3"
)

func newInitCommand() *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: "Generate age key pair and register public key to GHCR",
		Action: runInit,
	}
}

func runInit(ctx context.Context, cmd *cli.Command) error {
	_ = cmd

	token, err := auth.LoadToken()
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("detecting repository: %w (run inside a git repository)", err)
	}

	// Check if already initialized
	dataDir := config.DataDir()
	if _, err := os.Stat(filepath.Join(dataDir, "age_key.enc")); err == nil {
		return fmt.Errorf("already initialized (age key exists at %s). Use --force to reinitialize", dataDir)
	}

	// Generate age key pair
	kp, err := age.GenerateKeyPair()
	if err != nil {
		return fmt.Errorf("generating age key pair: %w", err)
	}
	fmt.Printf("Generated age public key: %s\n", kp.PublicKey)

	// Token-lock the private key and save
	encrypted, err := tokenlock.Encrypt([]byte(kp.Identity.String()), token.AccessToken)
	if err != nil {
		return fmt.Errorf("encrypting private key: %w", err)
	}

	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return fmt.Errorf("creating data directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "age_key.enc"), encrypted, 0o600); err != nil {
		return fmt.Errorf("saving encrypted key: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "age_key.pub"), []byte(kp.PublicKey), 0o644); err != nil {
		return fmt.Errorf("saving public key: %w", err)
	}

	// Push public key to GHCR
	ref := fmt.Sprintf("ghcr.io/%s/enbu-recipients:%s", cfg.Owner, token.Username)
	fmt.Printf("Pushing public key to %s...\n", ref)
	pushOpts := &oci.PushOptions{
		SourceRepo: fmt.Sprintf("https://github.com/%s/%s", cfg.Owner, cfg.Repo),
	}
	if err := oci.Push(ctx, ref, "application/vnd.enbu.recipient.age.v1", []byte(kp.PublicKey), token.AccessToken, pushOpts); err != nil {
		return fmt.Errorf("pushing public key to GHCR: %w", err)
	}

	fmt.Println("Done! You are now registered.")
	return nil
}
