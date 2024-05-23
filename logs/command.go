package logs

import (
	"context"
	"flag"
	"fmt"
	"os"
)

const logsCommandName = "logs"

type command struct {
	fs     *flag.FlagSet
	client *Client
	opts   *Options
}

func NewLogsCommand() *command {
	cmd := &command{
		fs:   flag.NewFlagSet(logsCommandName, flag.ContinueOnError),
		opts: &Options{},
	}

	cmd.fs.Usage = func() {
		fmt.Printf("  %36s\n", "logs - command-line search for SolarWinds Observability log management service")
		fmt.Printf("    %2s, %16s %70s\n", "-h", "--help", "Show usage")
		fmt.Printf("    %2s  %16s %70s\n", "", "--count NUMBER", "Number of log entries to search (100)")
		fmt.Printf("    %2s  %16s %70s\n", "", "--min-time MIN", "Earliest time to search from")
		fmt.Printf("    %2s  %16s %70s\n", "", "--max-time MAX", "Latest time to search from")
		fmt.Printf("    %2s, %16s %70s\n", "-c", "--configfile", "Path to config (~/.swo-cli.yaml)")
		fmt.Printf("    %2s, %16s %70s\n", "-g", "--group GROUP_ID", "Group ID to search")
		fmt.Printf("    %2s, %16s %70s\n", "-s", "--system SYSTEM", "System to search")
		fmt.Printf("    %2s, %16s %70s\n", "-j", "--json", "Output raw JSON data (off)")
		fmt.Printf("    %2s, %16s %70s\n", "-V", "--version", "Display the version and exit")

		fmt.Println()

		fmt.Println("    Usage:")
		fmt.Println("      swo-cli logs [--min-time time] [--max-time time] [-g group-id] [-s system]")
		fmt.Println("        [-c swo-cli.yml] [-j] [--] [query]")

		fmt.Println()

		fmt.Println("    Examples:")
		fmt.Printf("    %s logs something\n", os.Args[0])
		fmt.Printf("    %s logs 1.2.3 Failure\n", os.Args[0])
		fmt.Printf(`    %s logs -s ns1 "connection refused"%v`, os.Args[0], "\n")
		fmt.Printf(`    %s logs "(www OR db) (nginx OR pgsql) -accepted"%v`, os.Args[0], "\n")
		fmt.Printf(`    %s logs -g <SWO_GROUP_ID> "(nginx OR pgsql) -accepted"%v`, os.Args[0], "\n")
		fmt.Printf(`    %s logs --min-time 'yesterday at noon' --max-time 'today at 4am' -g <SWO_GROUP_ID>%v`, os.Args[0], "\n")
		fmt.Printf("    %s logs -- -redis\n", os.Args[0])
	}

	cmd.fs.UintVar(&cmd.opts.count, "count", defaultCount, "")
	cmd.fs.StringVar(&cmd.opts.configFile, "c", "", "")
	cmd.fs.StringVar(&cmd.opts.configFile, "configfile", defaultConfigFile, "")
	cmd.fs.StringVar(&cmd.opts.group, "g", "", "")
	cmd.fs.StringVar(&cmd.opts.group, "group", "", "")
	cmd.fs.StringVar(&cmd.opts.system, "s", "", "")
	cmd.fs.StringVar(&cmd.opts.system, "system", "", "")
	cmd.fs.StringVar(&cmd.opts.ApiUrl, "api-url", defaultApiUrl, "")
	cmd.fs.StringVar(&cmd.opts.minTime, "min-time", "", "")
	cmd.fs.StringVar(&cmd.opts.maxTime, "max-time", "", "")
	cmd.fs.BoolVar(&cmd.opts.json, "j", false, "")
	cmd.fs.BoolVar(&cmd.opts.json, "json", false, "")
	cmd.fs.BoolVar(&cmd.opts.version, "V", false, "")
	cmd.fs.BoolVar(&cmd.opts.version, "version", false, "")

	return cmd
}

func (c *command) Init(args []string) error {
	err := c.fs.Parse(args)
	if err != nil {
		return err
	}

	opts, err := c.opts.Init(c.fs.Args())
	if err != nil {
		return err
	}

	client, err := NewClient(opts)
	if err != nil {
		return err
	}

	c.client = client

	return nil
}

func (c *command) Run(ctx context.Context) error {
	if c.client == nil {
		return fmt.Errorf("%s command was not initialized", logsCommandName)
	}

	return c.client.Run(ctx)
}

func (c *command) Name() string {
	return logsCommandName
}

func (c *command) Usage() {
	c.fs.Usage()
}
