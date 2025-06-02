package main

import (
	"os"

	"github.com/ghetzel/cli"
	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/orchestra"
)

func main() {
	var app = cli.NewApp()
	app.Name = orchestra.ApplicationName
	app.Usage = orchestra.ApplicationSummary
	app.Version = orchestra.ApplicationVersion
	app.EnableBashCompletion = true

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   `log-level, L`,
			Usage:  `Level of log output verbosity`,
			Value:  `info`,
			EnvVar: `LOGLEVEL`,
		},
		cli.StringFlag{
			Name:  `config, c`,
			Usage: `The name of the configuration file to load (if present)`,
			Value: orchestra.ConfigFile,
		},
	}

	app.Action = func(c *cli.Context) {
		orchestra.ConfigFile = c.String(`config`)
		log.FatalIf(orchestra.LoadDefaultConfig())

		log.FatalIf(
			orchestra.NewServer(
				orchestra.DefaultConfig,
			).ListenAndServe(),
		)
	}

	app.Run(os.Args)
}
