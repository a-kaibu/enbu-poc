package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/a-kaibu/enbu-poc/internal/age"
	"github.com/a-kaibu/enbu-poc/internal/auth"
	"github.com/a-kaibu/enbu-poc/internal/bundle"
	"github.com/a-kaibu/enbu-poc/internal/config"
	"github.com/a-kaibu/enbu-poc/internal/oci"
	"github.com/a-kaibu/enbu-poc/internal/tokenlock"
	"github.com/urfave/cli/v3"

	agecrypto "filippo.io/age"
)

func newPullCommand() *cli.Command {
	return &cli.Command{
		Name:  "pull",
		Usage: "Pull and decrypt the .env bundle",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "env",
				Usage: "Target environment",
				Value: "default",
			},
			&cli.BoolFlag{
				Name:  "stdout",
				Usage: "Output to stdout instead of .env file",
			},
		},
		Action: runPull,
	}
}

func runPull(ctx context.Context, cmd *cli.Command) error {
	env := cmd.String("env")
	toStdout := cmd.Bool("stdout")

	token, err := auth.LoadToken()
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// 1. Pull encrypted bundle from GHCR
	ref := fmt.Sprintf("ghcr.io/%s/enbu-bundle:%s", cfg.Owner, env)
	fmt.Fprintf(os.Stderr, "Pulling bundle from %s...\n", ref)

	ciphertext, err := oci.Pull(ctx, ref, token.AccessToken)
	if err != nil {
		return fmt.Errorf("pulling bundle: %w", err)
	}

	// 2. TODO: Verify cosign signature
	// For POC, skip signature verification

	// 3. Unlock age private key
	dataDir := config.DataDir()
	encKey, err := os.ReadFile(filepath.Join(dataDir, "age_key.enc"))
	if err != nil {
		return fmt.Errorf("reading encrypted key: %w (run 'enbu auth' first)", err)
	}

	privKeyBytes, err := tokenlock.Decrypt(encKey, token.AccessToken)
	if err != nil {
		return fmt.Errorf("unlocking private key: %w", err)
	}

	identity, err := agecrypto.ParseX25519Identity(string(privKeyBytes))
	if err != nil {
		return fmt.Errorf("parsing private key: %w", err)
	}

	// 4. Decrypt bundle
	plaintext, err := age.Decrypt(ciphertext, identity)
	if err != nil {
		return fmt.Errorf("decrypting bundle: %w", err)
	}

	// 5. Parse and output
	secrets, err := bundle.Unmarshal(plaintext)
	if err != nil {
		return fmt.Errorf("parsing bundle: %w", err)
	}

	dotenv := bundle.ToDotEnv(secrets)

	if toStdout {
		os.Stdout.Write(dotenv)
		return nil
	}

	if err := os.WriteFile(".env", dotenv, 0o600); err != nil {
		return fmt.Errorf("writing .env: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Written .env (%d secrets)\n", len(secrets))
	return nil
}
