package diecast

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/rxutil"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/typeutil"
	"github.com/oliveagle/jsonpath"
)

// [type=filter] Filter the incoming data using a JSONPath or regular expression.
// steps:
// - type: filter
//   data: ''
//
// -------------------------------------------------------------------------------------------------
type FilterStep struct{}

func (self *FilterStep) Perform(config *StepConfig, w http.ResponseWriter, req *http.Request, prev *StepConfig) (interface{}, error) {
	var lang string
	var expr string
	var negate bool

	data := prev.Output

	config.logstep("prev=%v input=%T", prev, data)

	if typeutil.IsMap(config.Data) {
		d := maputil.M(config.Data)
		lang = d.String(`language`)
		expr = d.String(`query`)
		negate = d.Bool(`negate`)
	} else {
		expr = typeutil.String(config.Data)
	}

	if expr != `` {
		switch lang {
		case `jsonpath`, ``:
			if res, err := jsonpath.JsonPathLookup(data, expr); err == nil {
				data = res
			} else {
				return nil, fmt.Errorf("jsonpath: %v", err)
			}
		case `regex`:
			if typeutil.IsScalar(data) {
				data = strings.Split(typeutil.String(data), "\n")
			}

			if typeutil.IsArray(data) {
				var out []string

				for _, line := range sliceutil.Stringify(data) {
					matches := rxutil.IsMatchString(expr, line)

					if (matches && !negate) || (!matches && negate) {
						out = append(out, line)
					}
				}
			} else {
				return nil, fmt.Errorf("filter language %q can only process strings and arrays of strings", `regex`)
			}
		default:
			return nil, fmt.Errorf("unrecognized filter language %q", lang)
		}
	}

	return data, nil
}
