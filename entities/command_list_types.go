package entities

import (
	"context"

	"github.com/solarwinds/swo-cli/config"
	"github.com/urfave/cli/v2"
)

func runListTypes(ctx *cli.Context) error {
	opts := NewOptions()
	opts.JSON = ctx.Bool("json")
	opts.Verbose = ctx.Bool(config.VerboseContextKey)
	opts.Token = ctx.String(config.TokenContextKey)
	opts.APIURL = ctx.String(config.APIURLContextKey)

	client, err := NewClient(opts)
	if err != nil {
		return err
	}

	return client.ListTypes(context.Background())
}
