package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/a-kaibu/enbu-poc/internal/age"
	"github.com/a-kaibu/enbu-poc/internal/auth"
	"github.com/a-kaibu/enbu-poc/internal/config"
	gh "github.com/a-kaibu/enbu-poc/internal/github"
	"github.com/a-kaibu/enbu-poc/internal/oci"
	"github.com/a-kaibu/enbu-poc/internal/tokenlock"
	"github.com/urfave/cli/v3"
)

const defaultClientID = "Ov23li6nFmfdF4FW9ikd"

func newAuthCommand() *cli.Command {
	return &cli.Command{
		Name:  "auth",
		Usage: "Authenticate with GitHub and generate age key pair",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "client-id",
				Usage:   "GitHub OAuth App client ID",
				Sources: cli.EnvVars("ENBU_CLIENT_ID"),
				Value:   defaultClientID,
			},
		},
		Action: runAuth,
	}
}

func runAuth(ctx context.Context, cmd *cli.Command) error {
	clientID := cmd.String("client-id")
	if clientID == "" {
		return fmt.Errorf("set ENBU_CLIENT_ID or --client-id flag with your GitHub OAuth App client ID")
	}

	// 1. OAuth Device Flow
	fmt.Println("Initiating GitHub authentication...")
	deviceResp, err := auth.RequestDeviceCode(ctx, clientID)
	if err != nil {
		return fmt.Errorf("requesting device code: %w", err)
	}

	fmt.Printf("\nOpen %s in your browser\n", deviceResp.VerificationURI)
	fmt.Printf("Enter code: %s\n\n", deviceResp.UserCode)
	fmt.Println("Waiting for authorization...")

	tokenResp, err := auth.PollForToken(ctx, clientID, deviceResp.DeviceCode, deviceResp.Interval)
	if err != nil {
		return fmt.Errorf("polling for token: %w", err)
	}

	// 2. Get username
	client := gh.NewClient(tokenResp.AccessToken)
	user, err := client.GetUser(ctx)
	if err != nil {
		return fmt.Errorf("getting user info: %w", err)
	}
	fmt.Printf("Authenticated as: %s\n", user.Login)

	// 3. Save token
	if err := auth.SaveToken(&auth.StoredToken{
		AccessToken: tokenResp.AccessToken,
		Username:    user.Login,
	}); err != nil {
		return fmt.Errorf("saving token: %w", err)
	}

	// 4. Generate age key pair
	kp, err := age.GenerateKeyPair()
	if err != nil {
		return fmt.Errorf("generating age key pair: %w", err)
	}
	fmt.Printf("Generated age public key: %s\n", kp.PublicKey)

	// 5. Token-lock the private key and save
	privateKeyStr := kp.Identity.String()
	encrypted, err := tokenlock.Encrypt([]byte(privateKeyStr), tokenResp.AccessToken)
	if err != nil {
		return fmt.Errorf("encrypting private key: %w", err)
	}

	dataDir := config.DataDir()
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return fmt.Errorf("creating data directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "age_key.enc"), encrypted, 0o600); err != nil {
		return fmt.Errorf("saving encrypted key: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "age_key.pub"), []byte(kp.PublicKey), 0o644); err != nil {
		return fmt.Errorf("saving public key: %w", err)
	}

	// 6. Push public key to GHCR
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Warning: could not detect repo (%v). Skipping GHCR push.\n", err)
		fmt.Println("Run 'enbu auth' again inside a git repository to push your public key.")
		return nil
	}

	ref := fmt.Sprintf("ghcr.io/%s/enbu-recipients:%s", cfg.Owner, user.Login)
	fmt.Printf("Pushing public key to %s...\n", ref)
	pushOpts := &oci.PushOptions{
		SourceRepo: fmt.Sprintf("https://github.com/%s/%s", cfg.Owner, cfg.Repo),
	}
	if err := oci.Push(ctx, ref, "application/vnd.enbu.recipient.age.v1", []byte(kp.PublicKey), tokenResp.AccessToken, pushOpts); err != nil {
		return fmt.Errorf("pushing public key to GHCR: %w", err)
	}

	fmt.Println("Done! You are now registered.")
	return nil
}
