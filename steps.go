package diecast

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/ghetzel/go-stockutil/executil"
	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/typeutil"
	"github.com/husobee/vestigo"
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
	if typeutil.IsArray(command) {
		if args := sliceutil.Stringify(command); len(args) > 0 {
			cmd = executil.Command(args[0], args[1:]...)
		} else {
			return nil, fmt.Errorf("Command array cannot be empty")
		}
	} else {
		cmd = executil.ShellCommand(typeutil.String(command))
	}

	if cmd != nil {
		cmd.Timeout = config.getTimeout()
		cmd.InheritEnv = inherit

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

		return cmd.Output()
	} else {
		return nil, fmt.Errorf("invalid shell")
	}
}

// [type=process] Process the output of the previous step by performing a sequence of discrete
//                operations on the data.
// -------------------------------------------------------------------------------------------------
type ProcessStep struct{}

func (self *ProcessStep) Perform(config *StepConfig, w http.ResponseWriter, req *http.Request, prev *StepConfig) (interface{}, error) {
	operations := sliceutil.Sliceify(config.Data)
	data := prev.Output

	config.logstep("prev=%v input=%T", prev, data)

	for _, o := range operations {
		operation := maputil.M(nil)
		var otype string

		if typeutil.IsMap(o) {
			operation = maputil.M(o)
			otype = operation.String(`do`)
		} else {
			otype = typeutil.String(o)
		}

		config.logstep("operation=%s", otype)

		switch otype {
		case `sort`, `rsort`:
			if typeutil.IsArray(data) {
				dataS := sliceutil.Sliceify(data)

				sort.Slice(dataS, func(i int, j int) bool {
					if otype == `rsort` {
						return typeutil.String(dataS[i]) > typeutil.String(dataS[j])
					} else {
						return typeutil.String(dataS[i]) < typeutil.String(dataS[j])
					}
				})

				data = dataS
			} else if data == nil {
				return make([]interface{}, 0), nil
			} else {
				return nil, fmt.Errorf("Can only sort arrays, got %T", data)
			}
		case `diffuse`:
			sep := operation.String(`separator`, `.`)
			joiner := operation.String(`joiner`, `=`)
			dataM := make(map[string]interface{})

			if typeutil.IsArray(data) {
				for _, item := range sliceutil.Sliceify(data) {
					if typeutil.IsScalar(item) {
						k, v := stringutil.SplitPair(typeutil.String(item), joiner)
						dataM[strings.TrimLeft(k, sep)] = typeutil.Auto(v)
					} else {

					}
				}
			} else if typeutil.IsMap(data) {
				dataM = maputil.M(data).MapNative()
			} else {
				return nil, fmt.Errorf("Can only diffuse arrays or maps, got %T", data)
			}

			if diffused, err := maputil.DiffuseMap(dataM, sep); err == nil {
				data = diffused
			} else {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("Unrecognized process operation %q", otype)
		}
	}

	return data, nil
}
