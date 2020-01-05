package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ghetzel/cli"
	"github.com/ghetzel/diecast"
	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/netutil"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/typeutil"
	yaml "gopkg.in/yaml.v2"
)

func main() {
	app := cli.NewApp()
	app.Name = diecast.ApplicationName
	app.Usage = diecast.ApplicationSummary
	app.Version = diecast.ApplicationVersion
	app.EnableBashCompletion = true

	server := diecast.NewServer(``)

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   `log-level, L`,
			Usage:  `Level of log output verbosity`,
			Value:  `info`,
			EnvVar: `LOGLEVEL`,
		},
		cli.StringFlag{
			Name:   `config, c`,
			Usage:  `The name of the configuration file to load (if present)`,
			Value:  diecast.DefaultConfigFile,
			EnvVar: `DIECAST_CONFIG`,
		},
		cli.StringFlag{
			Name:   `render`,
			Usage:  `Name a single path to render as a template, then exit.`,
			EnvVar: `DIECAST_RENDER_FILE`,
		},
		cli.StringFlag{
			Name:   `env, e`,
			Usage:  `The name of the environment.  This is used to load environment-specific configurations.`,
			EnvVar: `DIECAST_ENV`,
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
			Name:  `page, p`,
			Usage: `A key=value pair that will be inserted into the global page object.`,
		},
		cli.StringSliceFlag{
			Name:  `override, o`,
			Usage: `A key=value pair that will be inserted into the global page object, overriding any prior values.`,
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
		cli.BoolFlag{
			Name:  `build-site, B`,
			Usage: `Traverse the current directory, rendering all files into a static site.`,
		},
		cli.StringFlag{
			Name:  `build-destination, d`,
			Usage: `The destination directory to put files in when rendering a static site.`,
			Value: `./_site`,
		},
		cli.BoolFlag{
			Name:  `disable-commands`,
			Usage: `Set this flag to disable processing of prestart and start commands.`,
		},
		cli.StringSliceFlag{
			Name:  `prestart-command`,
			Usage: `Execute a command before starting the built-in web server.`,
		},
		cli.StringSliceFlag{
			Name:  `start-command`,
			Usage: `Execute a command before immediately after starting the built-in web server.`,
		},
		cli.BoolFlag{
			Name:  `debug, D`,
			Usage: `Allow template debugging by appending the "?__viewsource=true" query string parameter.`,
		},
		cli.BoolFlag{
			Name:  `autoindex, A`,
			Usage: `Allow directory listings to be automatically generated in the absence of an index file.`,
		},
		cli.BoolFlag{
			Name:  `help-functions`,
			Usage: `Generate documentation on all supported functions.`,
		},
		cli.BoolFlag{
			Name:  `tls`,
			Usage: `Start a TLS server instead of plain HTTP.`,
		},
		cli.StringFlag{
			Name:  `tls-crt, k`,
			Usage: `The certificate file for SSL/TLS.`,
			Value: `server.crt`,
		},
		cli.StringFlag{
			Name:  `tls-key, K`,
			Usage: `The certificate key file for SSL/TLS.`,
			Value: `server.key`,
		},
		cli.StringFlag{
			Name:  `tls-client-mode`,
			Usage: `Enable TLS client certificate validation; may be one of "request", "any", "verify", or "require"`,
		},
		cli.StringFlag{
			Name:  `tls-client-ca`,
			Usage: `Specify the path to the PEM-encoded certificate use to verify clients.`,
			Value: `clients.crt`,
		},
	}

	app.Before = func(c *cli.Context) error {
		if c.Bool(`help-functions`) {
			defs, _ := diecast.GetFunctions(nil)

			for _, group := range defs {
				if group.Description == `` {
					log.Warningf("%v: undocumented group", group.Name)
				}

				for _, fn := range group.Functions {
					if fn.Hidden {
						continue
					}

					if fn.Summary == `` {
						log.Warningf("%v: undocumented function", fn.Name)
					} else if len(fn.Examples) == 0 {
						log.Noticef("%v: no examples", fn.Name)
					} else if fn.Function != nil {
						if i, _, err := typeutil.FunctionArity(fn.Function); err == nil {
							if l := len(fn.Arguments); l < i {
								log.Noticef("%v: missing argdocs; have %d, expected %d", fn.Name, l, i)
							} else if l > i {
								log.Noticef("%v: too many argdocs; have %d, expected %d", fn.Name, l, i)
							}
						} else {
							log.Errorf("%v: %v", fn.Name, err)
						}
					}
				}
			}

			if data, err := json.MarshalIndent(&defs, ``, `  `); err == nil {
				os.Stdout.Write(data)
				os.Exit(0)
				return nil
			} else {
				return err
			}
		}

		log.SetLevelString(c.String(`log-level`))
		return nil
	}

	app.Action = func(c *cli.Context) {
		servePath := filepath.Clean(c.Args().First())

		server.RootPath = servePath
		server.BinPath, _ = filepath.Abs(os.Args[0])
		server.Address = c.String(`address`)
		server.Environment = c.String(`env`)
		server.EnableDebugging = c.Bool(`debug`)
		server.BindingPrefix = c.String(`binding-prefix`)
		server.RoutePrefix = c.String(`route-prefix`)
		server.TryLocalFirst = c.Bool(`local-first`)
		server.VerifyFile = c.String(`verify-file`)
		server.IndexFile = c.String(`index-file`)
		server.Autoindex = c.Bool(`autoindex`)
		server.DisableCommands = c.Bool(`disable-commands`)

		if c.IsSet(`tls`) {
			server.TLS = &diecast.TlsConfig{
				Enable:         c.Bool(`tls`),
				CertFile:       c.String(`tls-crt`),
				KeyFile:        c.String(`tls-key`),
				ClientCertMode: c.String(`tls-client-mode`),
				ClientCAFile:   c.String(`tls-client-ca`),
			}
		}

		for _, cmdline := range c.StringSlice(`prestart-command`) {
			if cmdline != `` {
				server.PrestartCommands = append(server.PrestartCommands, &diecast.StartCommand{
					Command: cmdline,
				})
			}
		}

		for _, cmdline := range c.StringSlice(`start-command`) {
			if cmdline != `` {
				server.StartCommands = append(server.StartCommands, &diecast.StartCommand{
					Command: cmdline,
				})
			}
		}

		populateFlags(server.DefaultPageObject, c.StringSlice(`page`))
		populateFlags(server.OverridePageObject, c.StringSlice(`override`))

		if err := server.LoadConfig(c.String(`config`)); err != nil {
			log.Fatalf("config error: %v", err)
		}

		if patterns := c.StringSlice(`template-pattern`); len(patterns) > 0 {
			if sliceutil.ContainsString(patterns, `none`) {
				server.TemplatePatterns = nil
			} else {
				server.TemplatePatterns = c.StringSlice(`template-pattern`)
			}
		}

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

		renderSingleFile := c.String(`render`)

		// is it hacky? sure.  but it works
		if renderSingleFile != `` {
			if abspath, err := filepath.Abs(renderSingleFile); err == nil {
				if port, err := netutil.EphemeralPort(); err == nil {
					server.RootPath = filepath.Dir(abspath)
					server.VerifyFile = filepath.Base(abspath)
					server.TemplatePatterns = append(server.TemplatePatterns, `/`+filepath.Base(abspath))
					server.Address = fmt.Sprintf("127.0.0.1:%d", port)
					server.BindingPrefix = fmt.Sprintf("http://%s", server.Address)
				} else {
					log.Fatalf("cannot allocate ephemeral port: %v", err)
				}
			} else {
				log.Fatalf("cannot get abspath: %v", err)
			}
		}

		if err := server.Initialize(); err == nil {
			scheme := `http`

			if ssl := server.TLS; ssl != nil && ssl.Enable {
				scheme = `https`
			}

			errchan := make(chan error)

			log.Infof("diecast v%v listening at %s://%s", diecast.ApplicationVersion, scheme, server.Address)

			go func() {
				errchan <- server.Serve()
			}()

			if c.Bool(`build-site`) {
				log.Infof("Rendering site in %v", servePath)
				paths := make([]string, 0)

				if err := filepath.Walk(servePath, func(path string, info os.FileInfo, err error) error {
					base := filepath.Base(path)
					ext := filepath.Ext(path)

					if strings.HasPrefix(base, `_`) {
						if info.IsDir() {
							return filepath.SkipDir
						}
					} else if strings.HasSuffix(strings.TrimSuffix(base, ext), `__id`) {
						return nil
					} else if !info.IsDir() {
						urlPath := strings.TrimPrefix(path, servePath)
						urlPath = strings.TrimPrefix(urlPath, `/`)
						urlPath = `/` + urlPath

						if !sliceutil.ContainsString(paths, urlPath) {
							paths = append(paths, urlPath)
						}
					}

					return nil
				}); err != nil {
					log.Fatalf("build error: %v", err)
				}

				destinationPath := c.String(`build-destination`)

				if err := os.RemoveAll(destinationPath); err != nil {
					log.Fatalf("Failed to cleanup destination: %v", err)
				}

				sort.Strings(paths)
				client := &http.Client{
					Timeout: time.Duration(10) * time.Second,
				}

				for _, path := range paths {
					response, err := client.Get(`http://` + server.Address + path)

					if err == nil && response.StatusCode >= 400 {
						err = fmt.Errorf("%v", response.Status)
					}

					if err == nil {
						destFile := filepath.Join(destinationPath, path)

						if err := os.MkdirAll(filepath.Dir(destFile), 0755); err != nil {
							log.Fatalf("Failed to create destination: %v", err)
						}

						if file, err := os.Create(destFile); err == nil {
							_, err := io.Copy(file, response.Body)

							if err != nil {
								log.Fatalf("Failed to write file %v: %v", destFile, err)
							}

							file.Close()
						} else {
							log.Fatalf("Failed to create file %v: %v", destFile, err)
						}
					} else {
						log.Fatalf("Request to %v failed: %v", path, err)
					}
				}
			} else {
				go func() {
					if renderSingleFile != `` {
						errchan <- server.RenderPath(os.Stdout, filepath.Base(renderSingleFile))
					}
				}()

				select {
				case err := <-errchan:
					log.FatalIf(err)
				}
			}
		} else {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}

	app.Run(os.Args)
}

func appendDataFile(data *maputil.Map, baseK string, filename string) {
	if filename == `` {
		return
	}

	if file, err := os.Open(filename); err == nil {
		defer file.Close()

		var parsed interface{}
		var err error

		ext := filepath.Ext(filename)
		ext = strings.ToLower(ext)

		switch ext {
		case `.yaml`:
			err = yaml.NewDecoder(file).Decode(&parsed)
		case `.txt`:
			pM := maputil.M(nil)

			if b, err := ioutil.ReadAll(file); err == nil {
				for _, line := range strings.Split(string(b), "\n") {
					line = strings.TrimSpace(line)

					if len(line) == 0 || strings.HasPrefix(line, `#`) {
						continue
					}

					k, v := stringutil.SplitPair(line, `=`)
					k = strings.TrimSpace(k)
					v = strings.TrimSpace(v)

					pM.Set(k, typeutil.Auto(v))
				}

				parsed = pM.MapNative()
			} else {
				log.Fatalf("bad data-file: %v", err)
			}
		case `.json`:
			err = json.NewDecoder(file).Decode(&parsed)
		default:
			return
		}

		if err != nil {
			log.Fatalf("bad data-file: %v", err)
		}

		log.Debugf("parsing data-file: path=%s key=%s", filename, baseK)

		if baseK != `` {
			data.Set(baseK, parsed)
		} else if typeutil.IsMap(parsed) {
			for k, v := range typeutil.MapNative(parsed) {
				data.Set(k, v)
			}
		} else if typeutil.IsArray(parsed) {
			for i, item := range sliceutil.Sliceify(parsed) {
				data.Set(typeutil.String(i), item)
			}
		}
	} else {
		log.Fatalf("bad data-file: %v", err)
	}
}

func populateFlags(into map[string]interface{}, from []string) {
	for _, pair := range from {
		key, value := stringutil.SplitPair(pair, `=`)
		maputil.DeepSet(into, strings.Split(key, `.`), stringutil.Autotype(value))
	}
}
