package entities

import (
	"context"

	"github.com/solarwinds/swo-cli/config"
	"github.com/urfave/cli/v2"
)

func runGet(ctx *cli.Context) error {
	opts := NewOptions()
	opts.ID = ctx.String("id")
	opts.JSON = ctx.Bool("json")
	opts.Verbose = ctx.Bool(config.VerboseContextKey)
	opts.Token = ctx.String(config.TokenContextKey)
	opts.APIURL = ctx.String(config.APIURLContextKey)

	if err := opts.ValidateForGet(); err != nil {
		return err
	}

	client, err := NewClient(opts)
	if err != nil {
		return err
	}

	return client.GetEntity(context.Background())
}
