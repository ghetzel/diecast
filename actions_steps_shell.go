package diecast

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/ghetzel/go-stockutil/executil"
	"github.com/ghetzel/go-stockutil/fileutil"
	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/typeutil"
	"github.com/husobee/vestigo"
	shellwords "github.com/mattn/go-shellwords"
)

// [type=shell] Return the output of a command as a response.
// Valid Configurations
//
// Command string passed to the user's $SHELL:
//
//	data: 'some shell command'
//
// Command array executed via exec*() system calls:
//
//	data: ['command', '--arg', 'x=1', '--help']
//
// Command options specified as an object:
//
//	data:
//		command:   				# interpreted the same as above (string or array)
// 		inherit: true|false 	# whether the current shell environment should be inherited by the command
//      env:
//			X: abc
//			Y: zyx
//
// -------------------------------------------------------------------------------------------------
type ShellStep struct{}

func (self *ShellStep) Perform(config *StepConfig, w http.ResponseWriter, req *http.Request, prev *StepConfig) (interface{}, error) {
	var cmd *executil.Cmd
	var command interface{}
	inherit := true
	env := make(map[string]interface{})

	// parse options format
	if typeutil.IsMap(config.Data) {
		cfg := maputil.M(config.Data)
		command = cfg.Get(`command`).Value
	} else {
		command = config.Data
	}

	// parse command line
	var args []string

	if typeutil.IsArray(command) {
		args = sliceutil.Stringify(command)
	} else {
		script := typeutil.String(command)

		// put multiline strings into a temp file and execute it as a standalone script
		if strings.Contains(script, "\n") {
			if shell := executil.FindShell(); shell != `` {
				if tmpfile, err := fileutil.WriteTempFile(script, `diecast-`); err == nil {
					config.logstep("multiline script written to %s", tmpfile)
					defer os.Remove(tmpfile)
					args = []string{shell, tmpfile}
				} else {
					return nil, fmt.Errorf("Failed write temporary file: %v", err)
				}
			} else {
				return nil, fmt.Errorf("Cannot locate user shell")
			}
		} else if a, err := shellwords.Parse(script); err == nil {
			args = a
		} else {
			return nil, fmt.Errorf("Failed to parse command line: %v", err)
		}
	}

	if len(args) > 0 {
		cmd = executil.Command(args[0], args[1:]...)
	} else {
		return nil, fmt.Errorf("Command array cannot be empty")
	}

	if cmd != nil {
		cmd.Timeout = config.getTimeout()
		cmd.InheritEnv = inherit
		cmd.Stdin = prev

		cmd.OnStart = func(s executil.Status) {
			config.logstep("command started (timeout: %v)", cmd.Timeout)
		}

		cmd.OnComplete = func(s executil.Status) {
			config.logstep("%v", s)
		}

		// explicitly-set environment variables
		for k, v := range env {
			cmd.SetEnv(k, v)
		}

		// request headers (prefixed with REQ_HEADER_)
		for k, v := range maputil.M(req.Header).MapNative() {
			k = stringutil.Underscore(k)
			k = strings.ToUpper(k)
			cmd.SetEnv(fmt.Sprintf("REQ_HEADER_%s", k), v)
		}

		// querystring params (prefixed with REQ_PARAM_)
		for k, v := range maputil.M(req.URL.Query()).MapNative() {
			k = strings.TrimPrefix(k, `:`)
			k = stringutil.Underscore(k)
			k = strings.ToUpper(k)
			cmd.SetEnv(fmt.Sprintf("REQ_PARAM_%s", k), v)
		}

		// positional URL parameters (prefixed with REQ_PARAM_)
		for _, k := range vestigo.ParamNames(req) {
			k = strings.TrimPrefix(k, `:`)
			kName := stringutil.Underscore(k)
			kName = strings.ToUpper(kName)
			cmd.SetEnv(fmt.Sprintf("REQ_PARAM_%s", kName), vestigo.Param(req, k))
		}

		return cmd.CombinedOutput()
	} else {
		return nil, fmt.Errorf("invalid shell")
	}
}
