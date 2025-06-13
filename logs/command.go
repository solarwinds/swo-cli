package logs

import (
	cli "github.com/urfave/cli/v2"
)

func NewLogsCommand() *cli.Command {
	return &cli.Command{
		Name:  "logs",
		Usage: "SolarWinds Observability logs",
		Subcommands: []*cli.Command{
			NewGetCommand(),
		},
	}
}
