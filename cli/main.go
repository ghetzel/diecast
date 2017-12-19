package main

import (
	"os"

	"github.com/ghetzel/cli"
	"github.com/ghetzel/diecast"
	"github.com/ghetzel/diecast/util"
	"github.com/op/go-logging"
)

var log = logging.MustGetLogger(`main`)

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
			Name:  `config, c`,
			Usage: `The name of the configuration file to load (if present)`,
			Value: diecast.DefaultConfigFile,
		},
		cli.StringFlag{
			Name:  `address, a`,
			Usage: `Address the HTTP server should listen on`,
			Value: diecast.DefaultAddress,
		},
		cli.StringFlag{
			Name:  `binding-prefix, b`,
			Usage: `The URL to be used for templates when resolving the loopback operator (:)`,
		},
		cli.StringFlag{
			Name:  `route-prefix`,
			Usage: `The path prepended to all HTTP requests`,
			Value: diecast.DefaultRoutePrefix,
		},
		cli.StringSliceFlag{
			Name:  `template-pattern, P`,
			Usage: `A shell glob pattern matching a set of files that should be templated`,
		},
		cli.StringSliceFlag{
			Name:  `mount, m`,
			Usage: `Expose a given as MOUNT and SOURCE when requested from the server (formatted as "MOUNT:SOURCE"; e.g. "/js:/usr/share/javascript")`,
		},
		cli.BoolTFlag{
			Name:  `local-first`,
			Usage: `Attempt to lookup files locally before evaluating mounts.`,
		},
		cli.StringFlag{
			Name:  `verify-file`,
			Usage: `Specifies a filename to verify the existence of (relative to the server root).`,
		},
		cli.StringFlag{
			Name:  `index-file`,
			Usage: `Specifies a default filename for paths ending in "/".`,
			Value: diecast.DefaultIndexFile,
		},
		cli.BoolFlag{
			Name:  `mounts-passthrough-requests, R`,
			Usage: `Whether to passthrough client requests to proxy mounts.`,
		},
		cli.BoolFlag{
			Name:  `mounts-passthrough-errors, E`,
			Usage: `Whether proxy mounts that return non 2xx HTTP statuses should be counted as valid responses.`,
		},
	}

	app.Before = func(c *cli.Context) error {
		level := logging.DEBUG

		if lvl, err := logging.LogLevel(c.String(`log-level`)); err == nil {
			level = lvl
		}

		logging.SetFormatter(logging.MustStringFormatter(`%{color}%{level:.4s}%{color:reset}[%{id:04d}] %{module}: %{message}`))
		logging.SetLevel(level, ``)

		return nil
	}

	app.Action = func(c *cli.Context) {
		server := diecast.NewServer(c.Args().First())

		server.Address = c.String(`address`)
		server.BindingPrefix = c.String(`binding-prefix`)
		server.RoutePrefix = c.String(`route-prefix`)
		server.TryLocalFirst = c.Bool(`local-first`)
		server.VerifyFile = c.String(`verify-file`)
		server.IndexFile = c.String(`index-file`)

		if err := server.LoadConfig(c.String(`config`)); err != nil {
			log.Fatalf("config error: %v", err)
		}

		server.TemplatePatterns = append(server.TemplatePatterns, c.StringSlice(`template-pattern`)...)

		mounts := make([]diecast.Mount, 0)

		for i, mountSpec := range c.StringSlice(`mount`) {
			if mount, err := diecast.NewMountFromSpec(mountSpec); err == nil {
				if proxyMount, ok := mount.(*diecast.ProxyMount); ok {
					proxyMount.PassthroughRequests = c.Bool(`mounts-passthrough-requests`)
					proxyMount.PassthroughErrors = c.Bool(`mounts-passthrough-errors`)

					if proxyMount.PassthroughRequests {
						log.Debugf("%T %d configured to passthrough client requests", proxyMount, i)
					}

					if proxyMount.PassthroughErrors {
						log.Debugf("%T %d configured to consider HTTP 4xx/5xx responses as valid", proxyMount, i)
					}
				}

				mounts = append(mounts, mount)
			}
		}

		server.SetMounts(mounts)

		for _, mount := range server.Mounts {
			log.Debugf("mount %T: %+v", mount, mount)
		}

		if err := server.Initialize(); err == nil {
			log.Infof("Starting HTTP server at http://%s", server.Address)

			if err := server.Serve(); err != nil {
				log.Fatal(err)
			}
		} else {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}

	app.Run(os.Args)
}
