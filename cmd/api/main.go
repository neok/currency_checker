package main

import (
	"fmt"
	"os"

	"github.com/neok/currency/internal/application"
	"github.com/neok/currency/internal/config"
	"github.com/neok/currency/internal/server"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "api:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.LoadAPI()

	app, cleanup, err := application.NewAPI(cfg)
	if err != nil {
		return err
	}
	defer cleanup()

	return server.Serve(app)
}
