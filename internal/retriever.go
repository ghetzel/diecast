package internal

import (
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
	// TODO: add Method and Headers{s}a to go-stockutil/fileutil.OpenOptions for http(s)

	var url = opts.URL

	for _, k := range maputil.StringKeys(opts.Params) {
		url = httputil.SetQString(url, k, opts.Params[k])
	}

	ctx.Debugf("  retrieve: %v", url)
	// self.Logf(
	// 	log.DEBUG,
	// 	"  ${cyan}\u25C0 HTTP %d %s${reset}; %d bytes; took %v; %d headers:",
	// 	code,
	// 	http.StatusText(code),
	// 	self.bytesWritten,
	// 	took.Round(time.Microsecond),
	// 	len(rhdr),
	// )

	if response, err := fileutil.OpenWithOptions(url, fileutil.OpenOptions{
		Timeout:  opts.Timeout,
		Insecure: opts.Insecure,
	}); err == nil {
		return io.NopCloser(response), nil
	} else {
		return nil, err
	}

	// dispatch request to protocol-specific retriever. result is io.ReadCloser
	//	.Method
	//	.URL
	//  .Insecure
	// 	.Headers (somehow inject originating request headers, global, and page-specific headers)
	// 	.Params
	// 	.Timeout

	// -> REQUEST
	// <- RESPONSE

	// return io.ReadCloser OR fallback; caller is responsible for parsing/filtering/transforming
}
