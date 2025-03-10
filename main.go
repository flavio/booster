package main

import (
	"fmt"
	"github.com/moio/booster/api"
	"os"

	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

// set by goreleaser via -ldflags at build time
// see https://golang.org/cmd/link/, https://goreleaser.com/customization/build/
// Empty string means snapshot build
var version string

func main() {
	app := cli.NewApp()
	app.Name = "booster"
	app.Usage = "Synchronizes container image registries efficiently"
	if version != "" {
		app.Version = version
	} else {
		app.Version = "snapshot"
	}
	app.EnableBashCompletion = true
	app.Action = serve

	app.Flags = []cli.Flag{
		cli.IntFlag{
			Name:  "port",
			Usage: "TCP port for the API (default 5000)",
			Value: 5000,
		},
		cli.StringFlag{
			Name:  "path",
			Usage: "path of the base registry directory (default /var/lib/registry)",
			Value: "/var/lib/registry",
		},
		cli.StringFlag{
			Name:  "primary",
			Usage: "http address of the primary, if any",
			Value: "",
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
}

func serve(ctx *cli.Context) error {
	path := ctx.String("path")
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.Errorf("%v is not a directory", path)
	}

	return api.Serve(path, ctx.Int("port"), ctx.String("primary"))
}
