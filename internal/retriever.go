package internal

import (
	"context"
	"io"
	"time"

	"github.com/ghetzel/go-stockutil/fileutil"
	"github.com/ghetzel/go-stockutil/httputil"
	"github.com/ghetzel/go-stockutil/maputil"
)

type RetrieveOptions struct {
	URL         string            `yaml:"url"                    json:"url"`                    // The URL or path to the resource being retrieved
	Headers     map[string]string `yaml:"headers,omitempty"      json:"headers,omitempty"`      // Additional headers to include in the request
	Insecure    bool              `yaml:"insecure,omitempty"     json:"insecure,omitempty"`     // If the protocol supports an insecure request mode (e.g.: HTTPS), permit it in this case
	Method      string            `yaml:"method,omitempty"       json:"method,omitempty"`       // The protocol-specific method to perform the request with
	ParamJoiner string            `yaml:"param_joiner,omitempty" json:"param_joiner,omitempty"` // If a parameter is provided as an array, but must be a string in the request, how shall the array elements be joined
	Params      map[string]any    `yaml:"params,omitempty"       json:"params,omitempty"`       // A set of additional parameters to include in the request (e.g.: HTTP query string parameters)
	Timeout     time.Duration     `yaml:"timeout"                json:"timeout"`                // Maximum time to wait for retrieval to complete before cancelling the request
	Isolated    bool              `yaml:"isolated,omitempty"     json:"isolated,omitempty"`     // Do not passthrough the headers that were sent from the client's browser in the originating request
}

func RetrieveData(ctx Contextable, opts *RetrieveOptions) (io.ReadCloser, error) {
	var url = opts.URL

	for _, k := range maputil.StringKeys(opts.Params) {
		url = httputil.SetQString(url, k, opts.Params[k])
	}

	// self.Logf(
	// 	log.DEBUG,
	// 	"  ${cyan}\u25C0 HTTP %d %s${reset}; %d bytes; took %v; %d headers:",
	// 	code,
	// 	http.StatusText(code),
	// 	self.bytesWritten,
	// 	took.Round(time.Microsecond),
	// 	len(rhdr),
	// )

	var retrieveCtx = context.Background()

	if tm := opts.Timeout; tm > 0 {
		var rctx, retrieveCancel = context.WithTimeout(retrieveCtx, tm)
		defer retrieveCancel()
		retrieveCtx = rctx
	}

	if opts.Insecure {
		retrieveCtx = context.WithValue(retrieveCtx, `insecure`, true)
	}

	if len(opts.Headers) > 0 {
		retrieveCtx = context.WithValue(retrieveCtx, `metadata`, opts.Headers)
	}

	if opts.Method != `` {
		retrieveCtx = context.WithValue(retrieveCtx, `method`, opts.Method)
	}

	if response, err := fileutil.Retrieve(retrieveCtx, url); err == nil {
		return response, nil
	} else {
		return nil, err
	}
}
