package diecast

import (
	"encoding/json"
	"fmt"
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
	Type     string      `json:"type"`
	Data     interface{} `json:"data"`
	Timeout  string      `json:"timeout,omitempty"`
	Parser   string      `json:"parser"`
	Output   interface{} `json:"-"`
	Error    error       `json:"-"`
	index    int
	firstlog bool
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
	if !self.firstlog {
		format = "\u2502  step %d: " + format
		args = append([]interface{}{self.index}, args...)
	} else {
		format = "\u2502          " + format
	}

	log.Debugf(format, args...)
	self.firstlog = true
}

type Action struct {
	Path   string        `json:"path"`
	Method interface{}   `json:"method"`
	Steps  []*StepConfig `json:"steps"`
}

// Performs the action in response to an HTTP request, evaluating all action steps.  Steps are
// responsible for generating and manipulating output.  The output of the last step will be returned,
// or an error will be returned if not nil.
func (self *Action) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	started := time.Now()

	prev := &StepConfig{
		Type:   `input`,
		Output: req.Body,
		index:  -1,
	}

	log.Debugf("\u256d %s %s", req.Method, req.URL.Path)

	for i, step := range self.Steps {
		step.index = i
		step.logstep("type=%v data=%T", step.Type, step.Data)
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
			httputil.RespondJSON(w, err)
		} else {
			httputil.RespondJSON(w, prev.Output)
		}
	} else {
		w.WriteHeader(http.StatusNoContent)
	}

	log.Debugf("\u2570 response sent (took: %v)", time.Since(started))
}
