package entities

import (
	"context"

	"github.com/urfave/cli/v2"
)

func runListTypes(ctx *cli.Context) error {
	opts := NewOptions()
	opts.JSON = ctx.Bool("json")
	opts.Verbose = ctx.Bool("verbose")
	opts.Token = ctx.String("api-token")
	opts.APIURL = ctx.String("api-url")

	client, err := NewClient(opts)
	if err != nil {
		return err
	}

	return client.ListTypes(context.Background())
}
