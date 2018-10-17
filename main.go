package main

import (
	"fmt"
	"github.com/pooflix/client"
	"github.com/pooflix/core"
	"github.com/urfave/cli"
	"net/url"
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

	c, err := core.NewDefaultClientConfig()
	if err != nil {
		panic(err)
	}

	app.Metadata = map[string]interface{}{
		"config": c,
	}

	app.Flags = []cli.Flag{

	}

	app.Commands = []cli.Command{
		{
			Name:    "client",
			Aliases: []string{"c"},
			Usage:   "add torrent",
			Action: func(ctx *cli.Context) error {
				cfg := ctx.App.Metadata["config"].(*core.Config)
				ip, err := core.GetLocalIp()
				if err != nil {
					return err
				}

				cl := client.NewClient(&url.URL{
					Host:   fmt.Sprintf("%s:%s", ip, cfg.HttpServerPort),
					Scheme: "http",
					Path:   "/api/v1",
				})
				torrents, err := cl.ListTorrents()
				if err != nil {
					return err
				}

				for _, t := range torrents {
					fmt.Println(t)
				}

				return nil
			},
		},
	}

	app.Action = func(ctx *cli.Context) error {
		//initialize core
		if err := core.New(c); err != nil {
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
