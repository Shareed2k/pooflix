package main

import (
	"fmt"
	"github.com/pooflix/core"
	"github.com/urfave/cli"
	"os"
	"runtime/debug"
)

var (
	// BuildTime is a time label of the moment when the binary was built
	BuildTime = "unset"
	// Commit is a last commit hash at the moment when the binary was built
	Commit = "unset"
	// Version is a semantic version of current build
	Version = "unversioned"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("panic recovered: %ss \n %s", r, debug.Stack())
			os.Exit(1)
		}
	}()

	app := cli.NewApp()
	app.Name = "PooFlix"
	app.Usage = "PooFlix " + Version
	app.Version = Version

	c := core.NewDefaultClientConfig()

	app.Metadata = map[string]interface{}{
		"config": c,
	}

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "download-dir, d",
			Usage:       "Download directory path",
			Destination: &c.DownloadDirectory,
			// Value:  <--- NOTE just cuz you always forget you can set defaults
		},
	}

	app.Commands = []cli.Command{
		{
			Name:    "client",
			Aliases: []string{"c"},
			Usage:   "add torrent",
			Action: func(ctx *cli.Context) error {
				cfg := ctx.App.Metadata["config"].(*core.Config)
				fmt.Println("from client, ", cfg)

				return nil
			},
		},
	}

	app.Action = func(ctx *cli.Context) error {
		_, err := core.New(c)
		if err != nil {
			return err
		}

		return nil
	}

	fmt.Printf(
		"%s\ncommit: %s, build time: %s, release: %s\n\n",
		app.Usage, Commit, BuildTime, Version,
	)

	if err := app.Run(os.Args); err != nil {
		panic(err)
	}
}
