package diecast

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/typeutil"
)

// [type=filter] Filter the incoming data using a JSONPath or regular expression.
// steps:
// - type: filter
//   data: ''
//
// -------------------------------------------------------------------------------------------------
type RequestStep struct{}

func (self *RequestStep) Perform(config *StepConfig, w http.ResponseWriter, req *http.Request, prev *StepConfig) (interface{}, error) {
	var url string
	var method string
	var protocol Protocol

	data := prev.Output

	config.logstep("prev=%v input=%T", prev, data)

	if typeutil.IsMap(config.Data) {
		d := maputil.M(config.Data)
		url = d.String(`url`)
		method = strings.ToUpper(d.String(`method`))
		negate = d.Bool(`negate`)
	} else {
		url = typeutil.String(config.Data)
	}

	if method == `` {
		method = http.MethodGet
	}

	if url == `` {
		return nil, fmt.Errorf("Must specify a URI")
	}

	if uri, err := url.Parse(url); err == nil {
		var protocol Protocol

		if p, ok := registeredProtocols[uri.Scheme]; ok && p != nil {
			protocol = p
		} else {
			return nil, fmt.Errorf("unsupported protocol %q", uri.Scheme)
		}

		if response, err := protocol.Retrieve(&ProtocolRequest{
			Verb:          method,
			URL:           uri,
			Request:       req,
			Binding:       nil,
			Header:        nil,
			TemplateData:  nil,
			TemplateFuncs: nil,
		}); err == nil {
			defer response.Close()
			return response.Decode()
		} else {
			return nil, fmt.Errorf("request failed: %v", err)
		}
	} else {
		return nil, fmt.Errorf("invalid url: %v", err)
	}
}
