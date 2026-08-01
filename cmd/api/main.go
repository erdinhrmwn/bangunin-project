// Command api runs the Bangunin HTTP API server.
package main

import (
	"context"
	"fmt"
	"os"

	"erdinhrmwn/bangunin/config"
	"erdinhrmwn/bangunin/internal/app"
)

func main() {
	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		fatal(err)
	}

	container, err := app.NewContainer(context.Background(), cfg)
	if err != nil {
		fatal(err)
	}
	defer container.Close()

	if err := container.Run(); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
