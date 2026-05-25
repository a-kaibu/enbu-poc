package cli

import (
	"context"
	"fmt"

	"github.com/a-kaibu/enbu-poc/internal/auth"
	gh "github.com/a-kaibu/enbu-poc/internal/github"
	"github.com/urfave/cli/v3"
)

const defaultClientID = "Ov23li6nFmfdF4FW9ikd"

func newAuthCommand() *cli.Command {
	return &cli.Command{
		Name:  "auth",
		Usage: "Authenticate with GitHub via OAuth Device Flow",
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

	client := gh.NewClient(tokenResp.AccessToken)
	user, err := client.GetUser(ctx)
	if err != nil {
		return fmt.Errorf("getting user info: %w", err)
	}

	if err := auth.SaveToken(&auth.StoredToken{
		AccessToken: tokenResp.AccessToken,
		Username:    user.Login,
	}); err != nil {
		return fmt.Errorf("saving token: %w", err)
	}

	fmt.Printf("Authenticated as: %s\n", user.Login)
	return nil
}
