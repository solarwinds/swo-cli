package entities

import (
	"github.com/urfave/cli/v2"

	"github.com/solarwinds/swo-cli/config"
)

func runList(ctx *cli.Context) error {
	opts := NewOptions()
	opts.Type = ctx.String("type")
	opts.Name = ctx.String("name")
	opts.JSON = ctx.Bool("json")
	opts.Verbose = ctx.Bool(config.VerboseContextKey)
	opts.Token = ctx.String(config.TokenContextKey)
	opts.APIURL = ctx.String(config.APIURLContextKey)

	if err := opts.ValidateForList(); err != nil {
		return err
	}

	client, err := NewClient(opts)
	if err != nil {
		return err
	}

	return client.ListEntities(ctx.Context)
}
