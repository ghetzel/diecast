package util

import (
	"fmt"
	"time"

	"github.com/ghetzel/cli"
)

const ApplicationName = `diecast`
const ApplicationSummary = `a dynamic site generator that consumes REST services and renders static HTML output in realtime`
const ApplicationVersion = `1.2.2`

var StartedAt = time.Now()
var SiSuffixes = []string{`bytes`, `KB`, `MB`, `GB`, `TB`, `PB`, `EB`, `YB`}

func Register() []cli.Command {
	return []cli.Command{
		{
			Name:  "version",
			Usage: "Output only the version string and exit",
			Action: func(c *cli.Context) {
				fmt.Println(ApplicationVersion)
			},
		},
	}
}
