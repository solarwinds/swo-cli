package entities

import (
	"context"

	"github.com/urfave/cli/v2"
)

func runList(ctx *cli.Context) error {
	opts := NewOptions()
	opts.Type = ctx.String("type")
	opts.Name = ctx.String("name")
	opts.PageSize = ctx.Int("page-size")
	opts.JSON = ctx.Bool("json")
	opts.Verbose = ctx.Bool("verbose")
	opts.Token = ctx.String("api-token")
	opts.APIURL = ctx.String("api-url")

	if err := opts.ValidateForList(); err != nil {
		return err
	}

	client, err := NewClient(opts)
	if err != nil {
		return err
	}

	return client.ListEntities(context.Background())
}
