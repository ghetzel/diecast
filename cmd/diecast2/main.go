package main

import (
	"io"
	"os"
	"path/filepath"

	"github.com/ghetzel/cli"
	"github.com/ghetzel/diecast/v2"
	"github.com/ghetzel/go-stockutil/executil"
	"github.com/ghetzel/go-stockutil/fileutil"
	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/typeutil"
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
				cli.StringFlag{
					Name:   `single-request, r`,
					Usage:  `Perform a single request against the given path and print the output.`,
					EnvVar: `DIECAST_SINGLE_REQUEST`,
				},
			},
			Action: func(c *cli.Context) {
				if sreq := c.String(`single-request`); sreq != `` {
					var method, path = stringutil.SplitPairTrailing(sreq, ` `)
					method = typeutil.OrString(method, `get`)

					if res, err := server.SimulateRequest(method, path, nil, nil, nil); err == nil {
						if res.Body != nil {
							defer res.Body.Close()
							io.Copy(os.Stdout, res.Body)
						}
					} else {
						log.Fatalf("request failed: %v", err)
					}
				} else {
					log.FatalIf(server.ListenAndServe(c.String(`address`)))
				}
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
