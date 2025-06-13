package main

import (
	"github.com/solarwinds/swo-cli/config"
	"log"
	"os"

	"github.com/solarwinds/swo-cli/logs"
	cli "github.com/urfave/cli/v2"
)

var version = "v1.3.2"

func main() {
	app := &cli.App{
		Name:    "swo",
		Usage:   "SolarWinds Observability Command-Line Interface",
		Version: version,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: config.APIURLContextKey, Usage: "URL of the SWO API", Value: config.DefaultAPIURL},
			&cli.StringFlag{Name: config.TokenContextKey, Usage: "API token"},
			&cli.StringFlag{Name: "config", Aliases: []string{"c"}, Usage: "path to config", Value: config.DefaultConfigFile},
		},
		Commands: []*cli.Command{
			logs.NewLogsCommand(),
		},
		Before: func(cCtx *cli.Context) error {
			cfg, err := config.Init(cCtx.String("config"), cCtx.String("api-url"), cCtx.String("api-token"))
			if err != nil {
				return err
			}
			if err = cCtx.Set(config.APIURLContextKey, cfg.APIURL); err != nil {
				return err
			}
			if err = cCtx.Set(config.TokenContextKey, cfg.Token); err != nil {
				return err
			}
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
