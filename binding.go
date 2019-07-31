package diecast

import (
	"bytes"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/typeutil"
)

var registeredProtocols = map[string]Protocol{
	``:      new(HttpProtocol),
	`http`:  new(HttpProtocol),
	`https`: new(HttpProtocol),
	`redis`: new(RedisProtocol),
}

// Register a new protocol handler that will handle URLs with the given scheme.
func RegisterProtocol(scheme string, protocol Protocol) {
	registeredProtocols[scheme] = protocol
}

type BindingErrorAction string

const (
	ActionSummarize BindingErrorAction = `summarize`
	ActionPrint                        = `print`
	ActionContinue                     = `continue`
	ActionBreak                        = `break`
	ActionIgnore                       = `ignore`
)

var BindingClient = &http.Client{
	Timeout: 60 * time.Second,
}

var AllowInsecureLoopbackBindings bool
var DefaultParamJoiner = `;`

type Binding struct {
	// The name of the key in the $.bindings template variable.
	Name string `json:"name,omitempty"`

	// Only evaluate the template on request URL paths matching one of the regular expressions in this array.
	Restrict []string `json:"restrict,omitempty"`

	// Only evaluate the binding if this expression yields a truthy value.
	OnlyIfExpr string `json:"only_if,omitempty"`

	// Do not evaluate the binding if this expression yields a truthy value.
	NotIfExpr string `json:"not_if,omitempty"`

	// The protocol-specific method to perform the request with.
	Method string `json:"method,omitempty"`

	// The URL that specifies the protocol and resource to retrieve.
	Resource string `json:"resource,omitempty"`

	// A duration specifying the timeout for the request.
	Timeout interface{} `json:"timeout,omitempty"`

	// If the protocol supports an insecure request mode (e.g.: HTTPS), permit it in this case.
	Insecure bool `json:"insecure,omitempty"`

	// A set of additional parameters to include in the request (e.g.: HTTP query string parameters)
	Params map[string]interface{} `json:"params,omitempty"`

	// If a parameter is provided as an array, but must be a string in the request, how shall the array elements be joined.
	ParamJoiner string `json:"param_joiner,omitempty"`

	// Additional headers to include in the request.
	Headers map[string]string `json:"headers,omitempty"`

	// If the request receives an open-ended body, this will allow structured data to be passed in.
	BodyParams map[string]interface{} `json:"body,omitempty"`

	// If the request receives an open-ended body, this will allow raw data to be passed in as-is.
	RawBody string `json:"rawbody,omitempty"`

	// How to serialize BodyParams into a string before the request is made.
	Formatter string `json:"formatter,omitempty"`

	// How to parse the response content from the request.
	Parser string `json:"parser,omitempty"`

	// Disable templating of variables in this binding.
	NoTemplate bool `json:"no_template,omitempty"`

	// Whether the request failing will cause a page-wide error or be ignored.
	Optional bool `json:"optional,omitempty"`

	// The value to place in $.bindings if the request fails.
	Fallback interface{} `json:"fallback,omitempty"`

	// Actions to take if the request fails.
	OnError BindingErrorAction `json:"on_error,omitempty"`

	// Actions to take in response to specific numeric response status codes.
	IfStatus map[int]BindingErrorAction `json:"if_status,omitempty"`

	// A templated value that yields an array.  The binding request will be performed once for each array element, wherein
	// the Resource value is passed into a template that includes the $index and $item variables, which represent the repeat
	// array item's position and value, respectively.
	Repeat string `json:"repeat,omitempty"`

	// Do not passthrough the headers that were sent to the template from the client's browser, even if Passthrough mode is enabled.
	SkipInheritHeaders bool `json:"skip_inherit_headers,omitempty"`

	// Reserved for future use.
	DisableCache bool `json:"disable_cache,omitempty"`

	// An open-ended set of options that are available for protocol implementations to use.
	ProtocolOptions map[string]interface{} `json:"protocol,omitempty"`
	server          *Server
}

func (self *Binding) ShouldEvaluate(req *http.Request) bool {
	if self.Restrict == nil {
		return true
	} else {
		for _, restrict := range self.Restrict {
			if rx, err := regexp.Compile(restrict); err == nil {
				if rx.MatchString(req.URL.Path) {
					return true
				}
			}
		}
	}

	return false
}

func (self *Binding) Evaluate(req *http.Request, header *TemplateHeader, data map[string]interface{}, funcs FuncMap) (interface{}, error) {
	id := reqid(req)
	log.Debugf("[%s] Evaluating binding %q", id, self.Name)

	if req.Header.Get(`X-Diecast-Binding`) == self.Name {
		return nil, fmt.Errorf("Loop detected")
	}

	method := strings.ToUpper(self.Method)
	uri := MustEvalInline(self.Resource, data, funcs)

	// bindings may specify that a request should be made to the currently server address by
	// prefixing the URL path with a colon (":") or slash ("/").
	//
	if strings.HasPrefix(uri, `:`) || strings.HasPrefix(uri, `/`) {
		var prefix string

		if self.server.BindingPrefix != `` {
			prefix = self.server.BindingPrefix
		} else {
			prefix = fmt.Sprintf("http://%s", req.Host)
		}

		prefix = strings.TrimSuffix(prefix, `/`)
		uri = strings.TrimPrefix(uri, `:`)
		uri = strings.TrimPrefix(uri, `/`)

		uri = fmt.Sprintf("%s/%s", prefix, uri)

		// allows bindings referencing the local server to avoid TLS cert verification
		// because the prefix is often `localhost:port`, which probably won't verify anyway.
		if AllowInsecureLoopbackBindings {
			self.Insecure = true
		}
	}

	if !self.NoTemplate {
		if self.OnlyIfExpr != `` {
			if v := MustEvalInline(self.OnlyIfExpr, data, funcs); typeutil.IsEmpty(v) || stringutil.IsBooleanFalse(v) {
				self.Optional = true
				return nil, fmt.Errorf("[%s] Binding %q not being evaluated because only_if expression was false", id, self.Name)
			}
		}

		if self.NotIfExpr != `` {
			if v := MustEvalInline(self.NotIfExpr, data, funcs); !typeutil.IsEmpty(v) && !stringutil.IsBooleanFalse(v) {
				self.Optional = true
				return nil, fmt.Errorf("[%s] Binding %q not being evaluated because not_if expression was truthy", id, self.Name)
			}
		}
	}

	if reqUrl, err := url.Parse(uri); err == nil {
		var protocol Protocol

		if p, ok := registeredProtocols[reqUrl.Scheme]; ok && p != nil {
			protocol = p
		} else {
			return nil, fmt.Errorf("Cannot evaluate binding %v: invalid protocol scheme %q", self.Name, reqUrl.Scheme)
		}

		log.Debugf("[%s]  binding %q: protocol=%T uri=%v", id, self.Name, protocol, uri)
		log.Infof("[%s] Binding: > %s %+v ? %s", id, strings.ToUpper(sliceutil.OrString(method, `get`)), reqUrl.String(), reqUrl.RawQuery)

		if response, err := protocol.Retrieve(&ProtocolRequest{
			Verb:          method,
			URL:           reqUrl,
			Binding:       self,
			Request:       req,
			Header:        header,
			TemplateData:  data,
			TemplateFuncs: funcs,
		}); err == nil {
			defer response.Close()

			onError := BindingErrorAction(MustEvalInline(string(self.OnError), data, funcs))

			// handle per-http-status response handlers
			if len(self.IfStatus) > 0 && response.StatusCode > 0 {
				// get the action for this code
				if statusAction, ok := self.IfStatus[response.StatusCode]; ok {
					statusAction = BindingErrorAction(MustEvalInline(string(statusAction), data, funcs))

					switch statusAction {
					case ActionIgnore:
						onError = ActionIgnore
					default:
						redirect := string(statusAction)

						if !self.NoTemplate {
							redirect = MustEvalInline(redirect, data, funcs)
						}

						// if a url or path was specified, redirect the parent request to it
						if strings.HasPrefix(redirect, `http`) || strings.HasPrefix(redirect, `/`) {
							return nil, RedirectTo(redirect)
						} else {
							return nil, fmt.Errorf("[%s] Invalid status action '%v'", id, redirect)
						}
					}
				}
			}

			if rv, err := response.DecodeAs(self.Parser); err == nil {
				return rv, nil
			} else {
				switch onError {
				case ActionPrint:
					if err != nil {
						return nil, fmt.Errorf("%v", err)
					} else {
						return nil, fmt.Errorf("%v", string(data[:]))
					}
				case ActionIgnore:
					return rv, nil
				default:
					redirect := string(onError)

					// if a url or path was specified, redirect the parent request to it
					if strings.HasPrefix(redirect, `http`) || strings.HasPrefix(redirect, `/`) {
						return nil, RedirectTo(redirect)
					} else {
						return nil, fmt.Errorf(
							"[%s] Request %s %v failed: %v",
							id,
							method,
							reqUrl,
							sliceutil.Or(err, response.StatusCode),
						)
					}
				}
			}
		} else {
			return nil, fmt.Errorf("[%s] HTTP %v", id, err)
		}
	} else {
		return nil, fmt.Errorf("[%s] url: %v", id, err)
	}
}

func MustEvalInline(input string, data map[string]interface{}, funcs FuncMap) string {
	if out, err := EvalInline(input, data, funcs); err == nil {
		return out
	} else {
		panic(err.Error())
	}
}

func EvalInline(input string, data map[string]interface{}, funcs FuncMap) (string, error) {
	tmpl := NewTemplate(`inline`, TextEngine)
	tmpl.Funcs(funcs)

	if err := tmpl.ParseString(input); err == nil {
		output := bytes.NewBuffer(nil)

		if err := tmpl.Render(output, data, ``); err == nil {
			// since this data may have been entity escaped by html/template, unescape it here
			return html.UnescapeString(output.String()), nil
		} else {
			return ``, fmt.Errorf("error evaluating %q: %v", input, err)
		}
	} else {
		return ``, err
	}
}
