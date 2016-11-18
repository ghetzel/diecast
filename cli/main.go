package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/ghetzel/diecast"
	"github.com/ghetzel/diecast/engines"
	"github.com/ghetzel/diecast/util"
	"os"
)

func main() {
	app := cli.NewApp()
	app.Name = util.ApplicationName
	app.Usage = util.ApplicationSummary
	app.Version = util.ApplicationVersion
	app.EnableBashCompletion = false

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   `log-level, L`,
			Usage:  `Level of log output verbosity`,
			Value:  `info`,
			EnvVar: `LOGLEVEL`,
		},
		cli.StringFlag{
			Name:   `config-file, c`,
			Usage:  `Path to the configuration file to use`,
			Value:  diecast.DEFAULT_CONFIG_PATH,
			EnvVar: `CONFIG_FILE`,
		},
		cli.StringFlag{
			Name:   `address, a`,
			Usage:  `Address the HTTP server should listen on`,
			Value:  diecast.DEFAULT_SERVE_ADDRESS,
			EnvVar: `HTTP_ADDR`,
		},
		cli.IntFlag{
			Name:   `port, p`,
			Usage:  `TCP port the HTTP server should listen on`,
			Value:  diecast.DEFAULT_SERVE_PORT,
			EnvVar: `HTTP_PORT`,
		},
		cli.StringFlag{
			Name:   `templates-dir, T`,
			Usage:  `Root path where templates are stored`,
			Value:  engines.DEFAULT_TEMPLATE_PATH,
			EnvVar: `TEMPLATES_DIR`,
		},
		cli.StringFlag{
			Name:   `static-dir, S`,
			Usage:  `Path where static assets are located`,
			Value:  diecast.DEFAULT_STATIC_PATH,
			EnvVar: `STATIC_PATH`,
		},
		cli.StringFlag{
			Name:   `route-prefix`,
			Usage:  `The path prepended to all HTTP requests`,
			Value:  diecast.DEFAULT_ROUTE_PREFIX,
			EnvVar: `ROUTE_PREFIX`,
		},
	}

	app.Before = func(c *cli.Context) error {
		util.ParseLogLevel(c.String(`log-level`))

		log.Infof("%s v%s started at %s", util.ApplicationName, util.ApplicationVersion, util.StartedAt)

		return nil
	}

	app.Action = func(c *cli.Context) {
		server := diecast.NewServer()

		server.Address = c.String(`address`)
		server.Port = c.Int(`port`)
		server.ConfigPath = c.String(`config-file`)
		server.StaticPath = c.String(`static-dir`)
		server.TemplatePath = c.String(`templates-dir`)
		server.RoutePrefix = c.String(`route-prefix`)

		if err := server.Initialize(); err == nil {
			log.Infof("Starting HTTP server at http://%s:%d", server.Address, server.Port)
			if err := server.Serve(); err != nil {
				log.Fatalf("Cannot start server: %v", err)
			}
		} else {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}

	app.Run(os.Args)
}
