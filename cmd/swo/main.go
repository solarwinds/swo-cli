package main

import (
	"github.com/solarwinds/swo-cli/logs"
	"github.com/urfave/cli/v2"
	"log"
	"os"
)

var version = "v1.1.1"

func main() {
	app := &cli.App{
		Name:    "swo",
		Usage:   "SolarWinds Observability Command-Line Interface",
		Version: version,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "api-url", Usage: "URL of the SWO API", Value: logs.DefaultApiUrl},
			&cli.StringFlag{Name: "api-token", Usage: "API token"},
			&cli.StringFlag{Name: "config", Aliases: []string{"c"}, Usage: "path to config", Value: logs.DefaultConfigFile},
		},
		Commands: []*cli.Command{
			logs.NewLogsCommand(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
