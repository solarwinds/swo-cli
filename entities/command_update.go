package entities

import (
	"context"

	"github.com/solarwinds/swo-cli/config"
	"github.com/urfave/cli/v2"
)

func runUpdate(ctx *cli.Context) error {
	opts := NewOptions()
	opts.ID = ctx.String("id")
	opts.JSON = ctx.Bool("json")
	opts.Verbose = ctx.Bool(config.VerboseContextKey)
	opts.Token = ctx.String(config.TokenContextKey)
	opts.APIURL = ctx.String(config.APIURLContextKey)

	// Parse tags
	tagStrings := ctx.StringSlice("tag")
	if err := opts.ParseTags(tagStrings); err != nil {
		return err
	}

	if err := opts.ValidateForUpdate(); err != nil {
		return err
	}

	client, err := NewClient(opts)
	if err != nil {
		return err
	}

	return client.UpdateEntity(context.Background())
}
