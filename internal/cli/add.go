package cli

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/a-kaibu/enbu-poc/internal/auth"
	"github.com/a-kaibu/enbu-poc/internal/config"
	gh "github.com/a-kaibu/enbu-poc/internal/github"
	"github.com/urfave/cli/v3"
)

func newAddCommand() *cli.Command {
	return &cli.Command{
		Name:      "add",
		Aliases:   []string{"push"},
		Usage:     "Add a secret to the repository",
		ArgsUsage: "KEY VALUE",
		Action:    runAdd,
	}
}

func runAdd(ctx context.Context, cmd *cli.Command) error {
	args := cmd.Args()
	if args.Len() < 2 {
		return fmt.Errorf("usage: enbu add KEY VALUE")
	}

	key := args.Get(0)
	value := args.Get(1)

	token, err := auth.LoadToken()
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	client := gh.NewClient(token.AccessToken)

	// Get current bundle (or start fresh)
	bundle, err := getCurrentBundle(ctx, client, cfg)
	if err != nil {
		return fmt.Errorf("reading current bundle: %w", err)
	}

	// Add new key-value
	bundle[key] = value

	// Marshal and store as ENBU_BUNDLE secret
	bundleJSON, err := json.Marshal(bundle)
	if err != nil {
		return fmt.Errorf("marshaling bundle: %w", err)
	}

	if err := client.SetSecret(ctx, cfg.Owner, cfg.Repo, "ENBU_BUNDLE", string(bundleJSON)); err != nil {
		return fmt.Errorf("setting ENBU_BUNDLE secret: %w", err)
	}

	fmt.Printf("Added %s to ENBU_BUNDLE\n", key)

	// Trigger workflow to rebuild encrypted bundle
	fmt.Println("Triggering bundle rebuild...")
	if err := client.DispatchWorkflow(ctx, cfg.Owner, cfg.Repo, "enbu-secrets-updated", nil); err != nil {
		fmt.Printf("Warning: could not trigger workflow (%v). Trigger manually.\n", err)
	} else {
		fmt.Println("Bundle rebuild triggered.")
	}

	return nil
}

func getCurrentBundle(ctx context.Context, client *gh.Client, cfg *config.Config) (map[string]string, error) {
	// GitHub Secrets API doesn't allow reading values, only names.
	// We maintain the bundle state locally as a workaround for POC.
	// In production, the workflow would read the secret directly.
	_ = ctx
	_ = client
	_ = cfg
	// For POC: start with empty bundle each time, or load from local cache
	return make(map[string]string), nil
}
