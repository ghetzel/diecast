package diecast

import (
	"net/http"

	"github.com/ghetzel/go-stockutil/maputil"
)

// [type=respond] Configure the HTTP response headers and status.
// -------------------------------------------------------------------------------------------------
type RespondStep struct{}

func (self *RespondStep) Perform(config *StepConfig, w http.ResponseWriter, req *http.Request, prev *StepConfig) (interface{}, error) {
	var opts = maputil.M(config.Data)
	var status = int(opts.Int(`status`))
	var data = prev.Output

	config.logstep("prev=%v input=%T", prev, data)

	if headers := opts.Map(`headers`); len(headers) > 0 {
		for k, v := range headers {
			w.Header().Set(k.String(), v.String())
		}
	}

	if redirect := opts.String(`redirect`); redirect != `` {
		w.Header().Set(`Location`, redirect)

		if status >= 300 && status < 400 {
			w.WriteHeader(status)
		} else {
			w.WriteHeader(http.StatusTemporaryRedirect)
		}
	} else if status > 0 {
		w.WriteHeader(status)
	}

	// TODO: check for 204/205 status and empty the body out

	return data, nil
}
