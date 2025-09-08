package entities

import (
	"github.com/urfave/cli/v2"
)

// NewEntitiesCommand creates the entities command
func NewEntitiesCommand() *cli.Command {
	return &cli.Command{
		Name:  "entities",
		Usage: "Retrieve and manage entities from SolarWinds Observability",
		Subcommands: []*cli.Command{
			{
				Name:   "list",
				Usage:  "List entities by type",
				Action: runList,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "type",
						Aliases:  []string{"t"},
						Usage:    "Filter entities by type (required)",
						Required: true,
					},
					&cli.StringFlag{
						Name:    "name",
						Aliases: []string{"n"},
						Usage:   "Filter entities by name",
					},
					&cli.BoolFlag{
						Name:    "json",
						Aliases: []string{"j"},
						Usage:   "Output in JSON format",
					},
				},
			},
			{
				Name:   "get",
				Usage:  "Get entity by ID",
				Action: runGet,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "id",
						Usage:    "Entity ID (required)",
						Required: true,
					},
					&cli.BoolFlag{
						Name:    "json",
						Aliases: []string{"j"},
						Usage:   "Output in JSON format",
					},
				},
			},
			{
				Name:   "update",
				Usage:  "Update entity tags",
				Action: runUpdate,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "id",
						Aliases:  []string{"id"},
						Usage:    "Entity ID (required)",
						Required: true,
					},
					&cli.StringSliceFlag{
						Name:    "tag",
						Aliases: []string{"t"},
						Usage:   "Tag in key=value format (can be specified multiple times)",
					},
					&cli.BoolFlag{
						Name:    "json",
						Aliases: []string{"j"},
						Usage:   "Output in JSON format",
					},
				},
			},
			{
				Name:   "list-types",
				Usage:  "List all available entity types",
				Action: runListTypes,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "json",
						Aliases: []string{"j"},
						Usage:   "Output in JSON format",
					},
				},
			},
		},
	}
}
