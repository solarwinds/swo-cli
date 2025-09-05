package entities

import (
	"context"

	"github.com/urfave/cli/v2"
)

func runUpdate(ctx *cli.Context) error {
	opts := NewOptions()
	opts.ID = ctx.String("id")
	opts.JSON = ctx.Bool("json")
	opts.Verbose = ctx.Bool("verbose")
	opts.Token = ctx.String("api-token")
	opts.APIURL = ctx.String("api-url")

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
