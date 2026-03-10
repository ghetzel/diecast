package main

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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
	app.Usage = diecast.ApplicationSummary
	app.Version = diecast.ApplicationVersion

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
		cli.BoolFlag{
			Name:  `build`,
			Usage: `Renders a tree of static files representing the current site.`,
		},
		cli.StringFlag{
			Name:  `build-sitemap`,
			Usage: `Uses the given sitemap.xml to detect which pages to generate (can be a filesystem path, URL, or rendered from Diecast directly.)`,
			Value: `/sitemap.xml`,
		},
		cli.StringFlag{
			Name:   `build-dest`,
			Usage:  `The directory where site build output will be placed.`,
			EnvVar: `DIECAST_BUILD_DEST`,
			Value:  `build`,
		},
		cli.StringSliceFlag{
			Name:  `build-include`,
			Usage: `An additional file or directory to include in the build output.`,
		},
		cli.StringSliceFlag{
			Name:  `build-exclude`,
			Usage: `A file glob pattern matching files to exclude in the build output.`,
		},
	}

	app.Before = func(c *cli.Context) error {
		log.SetLevelString(c.String(`log-level`))
		server = prepServer(c)
		return nil
	}

	app.Action = func(c *cli.Context) {
		var specs = c.Args()

		if len(specs) == 0 {
			specs = []string{`.`}
		}

		log.FatalIf(server.LoadLayersFromString(specs...))

		if c.Bool(`build`) {
			var buildFrom = c.String(`build-sitemap`)
			var sitemapReader io.ReadCloser

			if fileutil.IsNonemptyFile(buildFrom) {
				if file, err := os.Open(buildFrom); err == nil {
					sitemapReader = file
				} else {
					log.Fatalf("sitemap file: %v", err)
				}
			} else if strings.HasPrefix(buildFrom, `http`) {
				if result, err := http.Get(buildFrom); err == nil {
					sitemapReader = result.Body
				} else {
					log.Fatalf("sitemap url: %v", err)
				}
			} else if res, err := server.SimulateRequest(``, buildFrom, nil, nil, nil); err == nil {
				sitemapReader = res.Body
			} else {
				log.Fatalf("sitemap route: %v", err)
			}

			if sitemapReader != nil {
				defer sitemapReader.Close()

				if sitemap, err := diecast.ParseSitemap(sitemapReader); err == nil {
					if err := diecast.BuildSite(server, sitemap, &diecast.BuildOptions{
						DestinationDir: c.String(`build-dest`),
						IncludePaths:   c.StringSlice(`build-include`),
						ExcludePaths:   c.StringSlice(`build-exclude`),
					}); err == nil {
						log.Notice("build successful")
						os.Exit(0)
					} else {
						log.Fatalf("build failed: %v", err)
					}
				} else {
					log.Fatalf("bad sitemap: %v", err)
				}
			} else {
				log.Fatalf("unable to build from sitemap %q", buildFrom)
			}
		} else {
			log.FatalIf(server.ListenAndServe(c.String(`address`)))
		}

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
