package internal

import (
	"fmt"
	"time"
)

type RetrieveOptions struct {
	URL         string            `yaml:"url"                    json:"url"`                    // The URL or path to the resource being retrieved
	Fallback    any               `yaml:"fallback,omitempty"     json:"fallback,omitempty"`     // The default value to return of the retrieval fails
	Headers     map[string]string `yaml:"headers,omitempty"      json:"headers,omitempty"`      // Additional headers to include in the request
	Insecure    bool              `yaml:"insecure,omitempty"     json:"insecure,omitempty"`     // If the protocol supports an insecure request mode (e.g.: HTTPS), permit it in this case
	Method      string            `yaml:"method,omitempty"       json:"method,omitempty"`       // The protocol-specific method to perform the request with
	ParamJoiner string            `yaml:"param_joiner,omitempty" json:"param_joiner,omitempty"` // If a parameter is provided as an array, but must be a string in the request, how shall the array elements be joined
	Params      map[string]any    `yaml:"params,omitempty"       json:"params,omitempty"`       // A set of additional parameters to include in the request (e.g.: HTTP query string parameters)
	Timeout     time.Duration     `yaml:"timeout"                json:"timeout"`                // Maximum time to wait for retrieval to complete before cancelling the request
	Isolated    bool              `yaml:"isolated,omitempty"     json:"isolated,omitempty"`     // Do not passthrough the headers that were sent from the client's browser in the originating request
}

func RetrieveData(ctx Contextable, opts *RetrieveOptions) (any, error) {
	return nil, fmt.Errorf("Not Implemented: RetrieveURL()")
}
