package util

import (
    "fmt"
    "os"
    "time"

    log "github.com/Sirupsen/logrus"
    "github.com/codegangsta/cli"
)

const ApplicationName     = `diecast`
const ApplicationSummary  = `a dynamic site generator that consumes REST services and renders static HTML output in realtime`
const ApplicationVersion  = `0.1.2`

var StartedAt  = time.Now()
var SiSuffixes = []string{ `bytes`, `KB`, `MB`, `GB`, `TB`, `PB`, `EB`, `YB` }


func Register() []cli.Command {
    return []cli.Command{
        {
            Name:        "version",
            Usage:       "Output only the version string and exit",
            Action:      func(c *cli.Context){
                fmt.Println(ApplicationVersion)
            },
        },
    }
}

func ParseLogLevel(logLevel string) {
    log.SetOutput(os.Stderr)
    log.SetFormatter(&log.TextFormatter{
        ForceColors: true,
    })

    switch logLevel {
    case `info`:
        log.SetLevel(log.InfoLevel)
    case `warn`:
        log.SetLevel(log.WarnLevel)
    case `error`:
        log.SetLevel(log.ErrorLevel)
    case `fatal`:
        log.SetLevel(log.FatalLevel)
    case `quiet`:
        log.SetLevel(log.PanicLevel)
    default:
        log.SetLevel(log.DebugLevel)
    }
}
