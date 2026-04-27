package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/neok/currency/internal/application"
	"github.com/neok/currency/internal/config"
	"github.com/neok/currency/internal/job"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "fetch:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.LoadFetch(os.Args[1:])
	if err != nil {
		return err
	}

	app, cleanup, err := application.NewFetch(cfg)
	if err != nil {
		return err
	}
	defer cleanup()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return job.Run(ctx, app.Logger, app.Fetcher, app.Store, cfg.Currencies)
}
