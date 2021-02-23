package main

import (
	"os"
	"path/filepath"

	"github.com/ghetzel/cli"
	"github.com/ghetzel/diecast/v2"
	"github.com/ghetzel/go-stockutil/executil"
	"github.com/ghetzel/go-stockutil/fileutil"
	"github.com/ghetzel/go-stockutil/log"
)

var userAwareConfigFile = filepath.Join(
	executil.RootOrString(`/etc`, `.`),
	diecast.DefaultConfigFilename,
)

func main() {
	var server *diecast.Server
	var app = cli.NewApp()
	app.Name = `diecast2`
	app.Usage = ``
	app.Version = `2.0.0a1`

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   `log-level, L`,
			Usage:  `Level of log output verbosity`,
			Value:  `debug`,
			EnvVar: `LOGLEVEL`,
		},
		cli.StringFlag{
			Name:   `config, c`,
			Usage:  `Path to the configuration file to use.`,
			EnvVar: `DIECAST_CONFIG`,
			Value:  userAwareConfigFile,
		},
	}

	app.Before = func(c *cli.Context) error {
		log.SetLevelString(c.String(`log-level`))
		server = prepServer(c)
		return nil
	}

	app.Commands = []cli.Command{
		{
			Name:  `server`,
			Usage: `Start an HTTP server and serve the configured site.`,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   `address, a`,
					Usage:  `The address the server will listen on.`,
					EnvVar: `DIECAST_ADDRESS`,
				},
			},
			Action: func(c *cli.Context) {
				log.FatalIf(server.ListenAndServe(c.String(`address`)))
			},
		},
	}

	app.Run(os.Args)
}

func prepServer(c *cli.Context) *diecast.Server {
	var cfgfile = fileutil.MustExpandUser(c.GlobalString(`config`))

	if cfgfile != `` {
		if srv, err := diecast.NewServerFromFile(cfgfile); err == nil {
			return srv
		} else if !os.IsNotExist(err) {
			log.Fatal(err)
		}
	}

	return new(diecast.Server)
}
