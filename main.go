package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	"github.com/solarwinds/swo-cli/logs"
)

type Command interface {
	Init([]string) error
	Run(ctx context.Context) error
	Name() string
	Usage()
}

func showUsage(cmds []Command) {
	fmt.Printf("\nUsage: %v %v\n\n", os.Args[0], "<command> [options]")
	fmt.Printf("Commands:\n")
	for _, cmd := range cmds {
		cmd.Usage()
	}
}

func main() {
	cmds := []Command{
		logs.NewLogsCommand(),
	}

	if len(os.Args[1:]) < 1 {
		showUsage(cmds)
		os.Exit(1)
	}

	for _, cmd := range cmds {
		if cmd.Name() == os.Args[1] {
			if err := cmd.Init(os.Args[2:]); err != nil {
				if errors.Is(err, flag.ErrHelp) {
					os.Exit(0)
				}

				slog.Error("Failed to initialize the command", slog.String("cmd", cmd.Name()), slog.String("error", err.Error()))
				os.Exit(1)
			}

			ctx, cancel := context.WithCancel(context.Background())
			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt)
			defer func() {
				signal.Stop(c)
				cancel()
			}()
			go func() {
				select {
				case <-c:
					cancel()
				case <-ctx.Done():
				}
			}()

			if err := cmd.Run(ctx); err != nil {
				slog.Error("Failed to run the command", slog.String("cmd", cmd.Name()), slog.String("error", err.Error()))
				os.Exit(1)
			}

			os.Exit(0)
		}
	}

	showUsage(cmds)
	os.Exit(1)
}
