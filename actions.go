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
	Perform(config *StepConfig, w http.ResponseWriter, req *http.Request, prev *StepConfig) (interface{}, error)
}

type StepConfig struct {
	Type     string      `yaml:"type"              json:"type"`              // The type of step
	Data     interface{} `yaml:"data"              json:"data"`              // The data being passed into this step from the previous one
	Timeout  string      `yaml:"timeout,omitempty" json:"timeout,omitempty"` // Timeout for this step
	Parser   string      `yaml:"parser"            json:"parser"`            // The format the data being passed in is expected to be in
	Output   interface{} `yaml:"-"                 json:"-"`
	Error    error       `yaml:"-"                 json:"-"`
	index    int
	firstlog bool
	reader   io.Reader
}

func (self *StepConfig) String() string {
	if t := self.Type; t == `` {
		return `(unknown)`
	} else {
		return t
	}
}

func (self *StepConfig) postprocess() {
	switch self.Parser {
	case ``, `json`:
		out := typeutil.Bytes(self.Output)

		if len(out) > 0 {
			if stringutil.IsSurroundedBy(out, `[`, `]`) {
				var outA []interface{}

				if err := json.Unmarshal(out, &outA); err == nil {
					self.Output = outA
				} else {
					self.Error = err
				}

			} else if stringutil.IsSurroundedBy(out, `[`, `]`) {
				var outM map[string]interface{}

				if err := json.Unmarshal(out, &outM); err == nil {
					self.Output = outM
				} else {
					self.Error = err
				}
			} else {
				var outI interface{}

				if err := json.Unmarshal(out, &outI); err == nil {
					self.Output = outI
				} else if log.ErrHasPrefix(err, `invalid character`) {
					lines := strings.Split(string(out), "\n")

					var outA []interface{}
					var asLines bool

					for i, line := range lines {
						var outM map[string]interface{}

						if err := json.Unmarshal([]byte(line), &outM); err == nil {
							outA = append(outA, outM)
						} else if log.ErrHasPrefix(err, `invalid character`) {
							asLines = true
							break
						} else {
							self.logstep("output line=%d: err=%v", i, err)
						}
					}

					if asLines {
						outA = sliceutil.Sliceify(sliceutil.CompactString(lines))
					}

					self.Output = outA
				} else {
					self.Error = err
				}
			}
		}

	case `lines`:
		self.Output = sliceutil.CompactString(
			strings.Split(typeutil.String(self.Output), "\n"),
		)

	default:
		self.Error = fmt.Errorf("Unsupported step parser %q", self.Parser)
		break
	}
}

func (self *StepConfig) Read(b []byte) (int, error) {
	if r, ok := self.Output.(io.Reader); ok {
		return r.Read(b)
	} else if self.reader == nil {
		if data, err := json.Marshal(self.Output); err == nil {
			self.reader = bytes.NewBuffer(data)
		} else {
			return 0, err
		}
	}

	return self.reader.Read(b)
}

func (self *StepConfig) Close() error {
	if self.reader != nil {
		self.reader = nil
	}

	if c, ok := self.Output.(io.Closer); ok {
		return c.Close()
	} else {
		return nil
	}
}

func (self *StepConfig) getTimeout() time.Duration {
	if self.Timeout != `` {
		if timeout, err := timeutil.ParseDuration(self.Timeout); err == nil {
			return timeout
		}
	}

	return DefaultActionStepTimeout
}

func (self *StepConfig) Perform(_ *StepConfig, w http.ResponseWriter, req *http.Request, prev *StepConfig) (interface{}, error) {
	if step, ok := steps[self.Type]; ok {
		return step.Perform(self, w, req, prev)
	} else {
		return nil, fmt.Errorf("Unrecognized action step type %q", self.Type)
	}
}

func (self *StepConfig) logstep(format string, args ...interface{}) {
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
	Method interface{}   `yaml:"method"         json:"method"`         // The HTTP method(s) this action will respond to
	Steps  []*StepConfig `yaml:"steps"          json:"steps"`          // The list of steps that are applied, in order, to the request body in order to generate a response
}

// Performs the action in response to an HTTP request, evaluating all action steps.  Steps are
// responsible for generating and manipulating output.  The output of the last step will be returned,
// or an error will be returned if not nil.
func (self *Action) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	started := time.Now()
	name := self.Name

	if name == `` {
		name = fmt.Sprintf("%s %s", req.Method, req.URL.Path)
	}

	var initData interface{}

	if req.ContentLength > 0 {
		defer req.Body.Close()

		var asMap map[string]interface{}

		if err := httputil.ParseRequest(req, &asMap); err == nil {
			initData = asMap
		} else {
			httputil.RespondJSON(w, err)
		}
	} else {
		initData = req.Body
	}

	prev := &StepConfig{
		Type:   `input`,
		Output: initData,
		index:  -1,
	}

	prev.postprocess()

	log.Debugf("\u256d Run action %s", name)

	for i, step := range self.Steps {
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
			httputil.RespondJSON(w, map[string]interface{}{
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
