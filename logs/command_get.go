package logs

import (
	"context"

	"github.com/solarwinds/swo-cli/config"
	"github.com/solarwinds/swo-cli/shared"
	cli "github.com/urfave/cli/v2"
)

// Context keys for command line flags
const (
	ConfigContextKey  = "config"
	GroupContextKey   = "group"
	SystemContextKey  = "system"
	MaxTimeContextKey = "max-time"
	MinTimeContextKey = "min-time"
	JSONContextKey    = "json"
	FollowContextKey  = "follow"
)

var flagsGet = []cli.Flag{
	&cli.StringFlag{Name: GroupContextKey, Aliases: []string{"g"}, Usage: "group name to search"},
	&cli.StringFlag{Name: MinTimeContextKey, Usage: "earliest time to search from", Value: "1 hour ago"},
	&cli.StringFlag{Name: MaxTimeContextKey, Usage: "latest time to search from"},
	&cli.StringFlag{Name: SystemContextKey, Aliases: []string{"s"}, Usage: "system to search"},
	&cli.BoolFlag{Name: JSONContextKey, Aliases: []string{"j"}, Usage: "output raw JSON", Value: false},
	&cli.BoolFlag{Name: FollowContextKey, Aliases: []string{"f"}, Usage: "enable live tailing", Value: false},
}

func runGet(cCtx *cli.Context) error {
	opts := &Options{
		args:       cCtx.Args().Slice(),
		configFile: cCtx.String(ConfigContextKey),
		group:      cCtx.String(GroupContextKey),
		system:     cCtx.String(SystemContextKey),
		maxTime:    cCtx.String(MaxTimeContextKey),
		minTime:    cCtx.String(MinTimeContextKey),
		json:       cCtx.Bool(JSONContextKey),
		follow:     cCtx.Bool(FollowContextKey),
		BaseOptions: shared.BaseOptions{
			Verbose: cCtx.Bool(config.VerboseContextKey),
			APIURL:  cCtx.String(config.APIURLContextKey),
			Token:   cCtx.String(config.TokenContextKey),
		},
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

// NewGetCommand creates a new 'logs get' command
func NewGetCommand() *cli.Command {
	return &cli.Command{
		Name:  "get",
		Usage: "command-line search for SolarWinds Observability log management service",
		Flags: flagsGet,
		ArgsUsage: `

EXAMPLES:
   swo logs get something
   swo logs get 1.2.3 Failure
   swo logs get -s ns1 "connection refused"
   swo logs get -f "(www OR db) (nginx OR pgsql) -accepted"
   swo logs get -f -g <SWO_GROUP_NAME> "(nginx OR pgsql) -accepted"
   swo logs get --min-time 'yesterday at noon' --max-time 'today at 4am' -g <SWO_GROUP_NAME>
   swo logs get -- -redis
`,
		Action: runGet,
	}
}
