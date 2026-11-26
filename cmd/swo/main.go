// Command line tool for SolarWinds Observability
package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/solarwinds/swo-cli/config"
	"github.com/solarwinds/swo-cli/entities"

	cli "github.com/urfave/cli/v2"

	"github.com/solarwinds/swo-cli/logs"
)

// Automatically updated by goreleaser in CI
var version = "v1.3.9"

func main() {
	app := &cli.App{
		Name:    "swo",
		Usage:   "SolarWinds Observability Command-Line Interface",
		Version: version,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: config.APIURLContextKey, Usage: "URL of the SWO API (must be https://)", Value: config.DefaultAPIURL},
			&cli.StringFlag{Name: config.TokenContextKey, Usage: "API token (not recommended: visible in process list and shell history; prefer the SWO_API_TOKEN env var or the config file)"},
			&cli.StringFlag{Name: "config", Aliases: []string{"c"}, Usage: "path to config", Value: config.DefaultConfigFile},
			&cli.BoolFlag{Name: "verbose", Usage: "enable verbose output (shows API URLs and debug info)"},
		},
		Commands: []*cli.Command{
			logs.NewLogsCommand(),
			entities.NewEntitiesCommand(),
		},
		Before: func(cCtx *cli.Context) error {
			// Only pass CLI values if they were explicitly set by the user
			var apiURL, apiToken string
			if cCtx.IsSet(config.APIURLContextKey) {
				apiURL = cCtx.String(config.APIURLContextKey)
			}
			if cCtx.IsSet(config.TokenContextKey) {
				apiToken = cCtx.String(config.TokenContextKey)
			}

			cfg, err := config.Init(cCtx.String("config"), cCtx.IsSet("config"), apiURL, apiToken)
			if err != nil {
				return err
			}
			if err = cCtx.Set(config.APIURLContextKey, cfg.APIURL); err != nil {
				return err
			}
			if err = cCtx.Set(config.TokenContextKey, cfg.Token); err != nil {
				return err
			}
			return nil
		},
	}

	ctx, received, stop := notifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := app.RunContext(ctx, os.Args)
	stop()

	if sig := received(); sig != nil && errors.Is(err, context.Canceled) {
		os.Exit(signalExitCode(sig))
	}
	if err != nil {
		log.Fatal(err)
	}
}

// notifyContext is like signal.NotifyContext but also reports which signal cancelled the context.
func notifyContext(parent context.Context, signals ...os.Signal) (ctx context.Context, received func() os.Signal, stop func()) {
	ctx, cancel := context.WithCancel(parent)
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, signals...)

	var mu sync.Mutex
	var got os.Signal
	done := make(chan struct{})
	go func() {
		select {
		case sig := <-ch:
			// Restore default handling so a second signal terminates immediately.
			signal.Stop(ch)
			mu.Lock()
			got = sig
			mu.Unlock()
			cancel()
		case <-done:
		}
	}()

	received = func() os.Signal {
		mu.Lock()
		defer mu.Unlock()
		return got
	}
	var once sync.Once
	stop = func() {
		once.Do(func() {
			signal.Stop(ch)
			close(done)
			cancel()
		})
	}
	return ctx, received, stop
}

// signalExitCode returns the shell convention exit status 128+n for signal n.
func signalExitCode(sig os.Signal) int {
	if s, ok := sig.(syscall.Signal); ok {
		return 128 + int(s)
	}
	return 1
}
