package logs

import (
	"context"

	"github.com/urfave/cli/v2"
)

var flags = []cli.Flag{
	&cli.StringFlag{Name: "group", Aliases: []string{"g"}, Usage: "group name to search"},
	&cli.StringFlag{Name: "min-time", Usage: "earliest time to search from", Value: "1 hour ago"},
	&cli.StringFlag{Name: "max-time", Usage: "latest time to search from"},
	&cli.StringFlag{Name: "system", Aliases: []string{"s"}, Usage: "system to search"},
	&cli.BoolFlag{Name: "json", Aliases: []string{"j"}, Usage: "output raw JSON", Value: false},
	&cli.BoolFlag{Name: "follow", Aliases: []string{"f"}, Usage: "enable live tailing", Value: false},
}

func run(cCtx *cli.Context) error {
	opts := &Options{
		args:       cCtx.Args().Slice(),
		configFile: cCtx.String("config"),
		group:      cCtx.String("group"),
		system:     cCtx.String("system"),
		maxTime:    cCtx.String("max-time"),
		minTime:    cCtx.String("min-time"),
		json:       cCtx.Bool("json"),
		follow:     cCtx.Bool("follow"),
		ApiUrl:     cCtx.String("api-url"),
		Token:      cCtx.String("api-token"),
	}
	if err := opts.Init(cCtx.Args().Slice()); err != nil {
		return err
	}
	client, err := NewClient(opts)
	if err != nil {
		return err
	}

	if err = client.Run(context.Background()); err != nil {
		return err
	}

	return nil
}

func NewLogsCommand() *cli.Command {
	return &cli.Command{
		Name:  "logs",
		Usage: "command-line search for SolarWinds Observability log management service",
		Flags: flags,
		ArgsUsage: `

EXAMPLES:
   swo logs something
   swo logs 1.2.3 Failure
   swo logs -s ns1 "connection refused"
   swo logs -f "(www OR db) (nginx OR pgsql) -accepted"
   swo logs -f -g <SWO_GROUP_NAME> "(nginx OR pgsql) -accepted"
   swo logs --min-time 'yesterday at noon' --max-time 'today at 4am' -g <SWO_GROUP_NAME>
   swo logs -- -redis
`,
		Action: run,
	}
}
