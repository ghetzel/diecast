package main

import (
    "os"
    log "github.com/Sirupsen/logrus"
    "github.com/codegangsta/cli"
    "github.com/ghetzel/diecast/util"
)

func main() {
    app                      := cli.NewApp()
    app.Name                  = util.ApplicationName
    app.Usage                 = util.ApplicationSummary
    app.Version               = util.ApplicationVersion
    app.EnableBashCompletion  = false

    app.Flags = []cli.Flag{
        cli.StringFlag{
            Name:   `log-level, L`,
            Usage:  `Level of log output verbosity`,
            Value:  `info`,
            EnvVar: `LOGLEVEL`,
        },
    }

    app.Before                = func(c *cli.Context) error {
        util.ParseLogLevel(c.String(`log-level`))

        log.Infof("%s v%s started at %s", util.ApplicationName, util.ApplicationVersion, util.StartedAt)

        return nil
    }


    app.Commands = []cli.Command{
        {
            Name:        `serve`,
            Usage:       `Start the HTTP server`,
            Flags:       []cli.Flag{
                cli.StringFlag{
                    Name:   `config-file, c`,
                    Usage:  `Path to the configuration file to use`,
                    Value:  DEFAULT_CONFIG_PATH,
                    EnvVar: `CONFIG_FILE`,
                },
                cli.StringFlag{
                    Name:   `templates-dir, T`,
                    Usage:  `Root path where templates are stored`,
                    Value:  DEFAULT_TEMPLATE_PATH,
                    EnvVar: `TEMPLATES_DIR`,
                },
                cli.StringFlag{
                    Name:   `address, a`,
                    Usage:  `Address the HTTP server should listen on`,
                    Value:  DEFAULT_SERVE_ADDRESS,
                    EnvVar: `HTTP_ADDR`,
                },
                cli.IntFlag{
                    Name:   `port, p`,
                    Usage:  `TCP port the HTTP server should listen on`,
                    Value:  DEFAULT_SERVE_PORT,
                    EnvVar: `HTTP_PORT`,
                },
            },
            Action:      func(c *cli.Context){
                server := NewServer()

                server.Address    = c.String(`address`)
                server.Port       = c.Int(`port`)
                server.ConfigPath = c.String(`config-file`)

                if err := server.Initialize(); err == nil {
                    log.Infof("Starting HTTP server at %s:%d", server.Address, server.Port)
                    server.Serve()
                }else{
                    log.Fatalf("Failed to start HTTP server: %v", err)
                }
            },
        },
    }

    app.Run(os.Args)
}
