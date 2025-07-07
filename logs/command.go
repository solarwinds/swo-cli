package logs

import (
	cli "github.com/urfave/cli/v2"
)

// NewLogsCommand creates a new 'logs' command
func NewLogsCommand() *cli.Command {
	return &cli.Command{
		Name:  "logs",
		Usage: "SolarWinds Observability logs",
		Subcommands: []*cli.Command{
			NewGetCommand(),
		},
	}
}
