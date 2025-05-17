package diecast

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ghetzel/go-stockutil/httputil"
	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/timeutil"
	"github.com/ghetzel/go-stockutil/typeutil"
)

var DefaultActionStepTimeout = 10 * time.Second

var steps = map[string]Performable{
	`shell`:   &ShellStep{},
	`process`: &ProcessStep{},
	`respond`: &RespondStep{},
}

// Register a performable step type to the given type name.
func RegisterActionStep(typeName string, performable Performable) {
	if performable != nil {
		steps[typeName] = performable
	} else {
		panic("cannot register nil step for type " + typeName)
	}
}

type Performable interface {
	Perform(config *StepConfig, w http.ResponseWriter, req *http.Request, prev *StepConfig) (any, error)
}

type StepConfig struct {
	Type    string `yaml:"type"              json:"type"`              // The type of step
	Data    any    `yaml:"data"              json:"data"`              // The data being passed into this step from the previous one
	Timeout string `yaml:"timeout,omitempty" json:"timeout,omitempty"` // Timeout for this step
	Parser  string `yaml:"parser"            json:"parser"`            // The format the data being passed in is expected to be in
	Output  any    `yaml:"-"                 json:"-"`
	Error   error  `yaml:"-"                 json:"-"`
	index   int
	reader  io.Reader
}

func (config *StepConfig) String() string {
	if t := config.Type; t == `` {
		return `(unknown)`
	} else {
		return t
	}
}

func (config *StepConfig) postprocess() {
	switch config.Parser {
	case ``, `json`:
		var out = typeutil.Bytes(config.Output)

		if len(out) > 0 {
			if stringutil.IsSurroundedBy(out, `[`, `]`) {
				var outA []any

				if err := json.Unmarshal(out, &outA); err == nil {
					config.Output = outA
				} else {
					config.Error = err
				}

			} else if stringutil.IsSurroundedBy(out, `[`, `]`) {
				var outM map[string]any

				if err := json.Unmarshal(out, &outM); err == nil {
					config.Output = outM
				} else {
					config.Error = err
				}
			} else {
				var outI any

				if err := json.Unmarshal(out, &outI); err == nil {
					config.Output = outI
				} else if log.ErrHasPrefix(err, `invalid character`) {
					var lines = strings.Split(string(out), "\n")

					var outA []any
					var asLines bool

					for i, line := range lines {
						var outM map[string]any

						if err := json.Unmarshal([]byte(line), &outM); err == nil {
							outA = append(outA, outM)
						} else if log.ErrHasPrefix(err, `invalid character`) {
							asLines = true
							break
						} else {
							config.logstep("output line=%d: err=%v", i, err)
						}
					}

					if asLines {
						outA = sliceutil.Sliceify(sliceutil.CompactString(lines))
					}

					config.Output = outA
				} else {
					config.Error = err
				}
			}
		}

	case `lines`:
		config.Output = sliceutil.CompactString(
			strings.Split(typeutil.String(config.Output), "\n"),
		)

	default:
		config.Error = fmt.Errorf("unsupported step parser %q", config.Parser)
	}
}

func (config *StepConfig) Read(b []byte) (int, error) {
	if r, ok := config.Output.(io.Reader); ok {
		return r.Read(b)
	} else if config.reader == nil {
		if data, err := json.Marshal(config.Output); err == nil {
			config.reader = bytes.NewBuffer(data)
		} else {
			return 0, err
		}
	}

	return config.reader.Read(b)
}

func (config *StepConfig) Close() error {
	if config.reader != nil {
		config.reader = nil
	}

	if c, ok := config.Output.(io.Closer); ok {
		return c.Close()
	} else {
		return nil
	}
}

func (config *StepConfig) getTimeout() time.Duration {
	if config.Timeout != `` {
		if timeout, err := timeutil.ParseDuration(config.Timeout); err == nil {
			return timeout
		}
	}

	return DefaultActionStepTimeout
}

func (config *StepConfig) Perform(_ *StepConfig, w http.ResponseWriter, req *http.Request, prev *StepConfig) (any, error) {
	if step, ok := steps[config.Type]; ok {
		return step.Perform(config, w, req, prev)
	} else {
		return nil, fmt.Errorf("unrecognized action step type %q", config.Type)
	}
}

func (config *StepConfig) logstep(format string, args ...any) {
	if format != `` {
		if !strings.HasPrefix(format, "\u2502") {
			format = "\u2502          " + format
		}

		log.Debugf(format, args...)
	}
}

type Action struct {
	Name   string        `yaml:"name,omitempty" json:"name,omitempty"` // The name of this action
	Path   string        `yaml:"path"           json:"path"`           // The URL path this action is accessible from
	Method any           `yaml:"method"         json:"method"`         // The HTTP method(s) this action will respond to
	Steps  []*StepConfig `yaml:"steps"          json:"steps"`          // The list of steps that are applied, in order, to the request body in order to generate a response
}

// Performs the action in response to an HTTP request, evaluating all action steps.  Steps are
// responsible for generating and manipulating output.  The output of the last step will be returned,
// or an error will be returned if not nil.
func (config *Action) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var started = time.Now()
	var name = config.Name

	if name == `` {
		name = fmt.Sprintf("%s %s", req.Method, req.URL.Path)
	}

	var initData any

	if req.ContentLength > 0 {
		defer req.Body.Close()

		var asMap map[string]any

		if err := httputil.ParseRequest(req, &asMap); err == nil {
			initData = asMap
		} else {
			httputil.RespondJSON(w, err)
		}
	} else {
		initData = req.Body
	}

	var prev = &StepConfig{
		Type:   `input`,
		Output: initData,
		index:  -1,
	}

	prev.postprocess()

	log.Debugf("\u256d Run action %s", name)

	for i, step := range config.Steps {
		step.index = i

		step.logstep("\u2502  step %d: type=%v data=%T", i, step.Type, step.Data)
		out, err := step.Perform(step, w, req, prev)

		prev = step
		prev.Output = out
		prev.Error = err
		prev.postprocess()
		step.logstep("output=%T err=%v", prev.Output, prev.Error)

		if prev.Error != nil && prev.Error.Error() == `stop` {
			step.logstep("break early", i)
			return
		}
	}

	if prev != nil {
		if err := prev.Error; err != nil {
			httputil.RespondJSON(w, map[string]any{
				`error`:  err.Error(),
				`output`: prev.Output,
			}, http.StatusInternalServerError)
		} else {
			httputil.RespondJSON(w, prev.Output)
		}
	} else {
		w.WriteHeader(http.StatusNoContent)
	}

	log.Debugf("\u2570 response sent (took: %v)", time.Since(started))
}
