package cli

import (
	"github.com/urfave/cli/v3"
)

func New(version string) *cli.Command {
	return &cli.Command{
		Name:    "enbu",
		Usage:   "Keyless .env management powered by GitHub",
		Version: version,
		Commands: []*cli.Command{
			newAuthCommand(),
			newAddCommand(),
			newPullCommand(),
		},
	}
}
