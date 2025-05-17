package diecast

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"html/template"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/ghetzel/go-stockutil/httputil"
	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/typeutil"
	"github.com/oliveagle/jsonpath"
	opentracing "github.com/opentracing/opentracing-go"
	"gopkg.in/yaml.v2"
)

var DefaultBindingTimeout = 60 * time.Second

var registeredProtocols = map[string]Protocol{
	``:           new(HttpProtocol),
	`http`:       new(HttpProtocol),
	`https`:      new(HttpProtocol),
	`http+unix`:  new(HttpProtocol),
	`https+unix`: new(HttpProtocol),
	`redis`:      new(RedisProtocol),
}

// Register a new protocol handler that will handle URLs with the given scheme.
func RegisterProtocol(scheme string, protocol Protocol) {
	registeredProtocols[scheme] = protocol
}

type BindingErrorAction string

const (
	ActionSummarize BindingErrorAction = `summarize`
	ActionPrint     BindingErrorAction = `print`
	ActionContinue  BindingErrorAction = `continue`
	ActionBreak     BindingErrorAction = `break`
	ActionIgnore    BindingErrorAction = `ignore`
)

var BindingClient = &http.Client{
	Timeout: 60 * time.Second,
}

var ErrSkipEval = errors.New(`skip evaluation`)
var AllowInsecureLoopbackBindings bool = true
var DefaultParamJoiner = `;`

type PaginatorConfig struct {
	Total        string            `yaml:"total"   json:"total"`
	Count        string            `yaml:"count"   json:"count"`
	Done         string            `yaml:"done"    json:"done"`
	Maximum      int64             `yaml:"max"     json:"max"`
	Data         string            `yaml:"data"    json:"data"`
	QueryStrings map[string]string `yaml:"params"  json:"params"`
	Headers      map[string]string `yaml:"headers" json:"headers"`
}

type ResultsPage struct {
	Page    int     `yaml:"page"           json:"page"`
	Last    bool    `yaml:"last,omitempty" json:"last,omitempty"`
	Range   []int64 `yaml:"range"          json:"range"`
	Data    any     `yaml:"data"           json:"data"`
	Counter int64   `yaml:"counter"        json:"counter"`
	Total   int64   `yaml:"total"          json:"total"`
}

type Binding struct {
	BodyParams         map[string]any                `yaml:"body,omitempty"                 json:"body,omitempty"`                 // If the request receives an open-ended body, this will allow structured data to be passed in.
	DisableCache       bool                          `yaml:"disable_cache,omitempty"        json:"disable_cache,omitempty"`        // Reserved for future use.
	Fallback           any                           `yaml:"fallback,omitempty"             json:"fallback,omitempty"`             // The value to place in $.bindings if the request fails.
	Formatter          string                        `yaml:"formatter,omitempty"            json:"formatter,omitempty"`            // How to serialize BodyParams into a string before the request is made.
	Headers            map[string]string             `yaml:"headers,omitempty"              json:"headers,omitempty"`              // Additional headers to include in the request.
	IfStatus           map[string]BindingErrorAction `yaml:"if_status,omitempty"            json:"if_status,omitempty"`            // Actions to take in response to specific numeric response status codes.
	Insecure           bool                          `yaml:"insecure,omitempty"             json:"insecure,omitempty"`             // If the protocol supports an insecure request mode (e.g.: HTTPS), permit it in this case.
	Method             string                        `yaml:"method,omitempty"               json:"method,omitempty"`               // The protocol-specific method to perform the request with.
	Name               string                        `yaml:"name,omitempty"                 json:"name,omitempty"`                 // The name of the key in the $.bindings template variable.
	NoTemplate         bool                          `yaml:"no_template,omitempty"          json:"no_template,omitempty"`          // Disable templating of variables in this binding.
	NotIfExpr          string                        `yaml:"not_if,omitempty"               json:"not_if,omitempty"`               // Do not evaluate the binding if this expression yields a truthy value.
	OnError            BindingErrorAction            `yaml:"on_error,omitempty"             json:"on_error,omitempty"`             // Actions to take if the request fails.
	OnlyIfExpr         string                        `yaml:"only_if,omitempty"              json:"only_if,omitempty"`              // Only evaluate the binding if this expression yields a truthy value.
	Optional           bool                          `yaml:"optional,omitempty"             json:"optional,omitempty"`             // Whether the request failing will cause a page-wide error or be ignored.
	Paginate           *PaginatorConfig              `yaml:"paginate,omitempty"             json:"paginate,omitempty"`             // A specialized repeater configuration that automatically performs pagination on an upstream request, aggregating the results before returning them.
	ParamJoiner        string                        `yaml:"param_joiner,omitempty"         json:"param_joiner,omitempty"`         // If a parameter is provided as an array, but must be a string in the request, how shall the array elements be joined.
	Params             map[string]any                `yaml:"params,omitempty"               json:"params,omitempty"`               // A set of additional parameters to include in the request (e.g.: HTTP query string parameters)
	Parser             string                        `yaml:"parser,omitempty"               json:"parser,omitempty"`               // How to parse the response content from the request.
	ProtocolOptions    map[string]any                `yaml:"protocol,omitempty"             json:"protocol,omitempty"`             // An open-ended set of options that are available for protocol implementations to use.
	RawBody            string                        `yaml:"rawbody,omitempty"              json:"rawbody,omitempty"`              // If the request receives an open-ended body, this will allow raw data to be passed in as-is.
	Repeat             string                        `yaml:"repeat,omitempty"               json:"repeat,omitempty"`               // A templated value that yields an array.  The binding request will be performed once for each array element, wherein the Resource value is passed into a template that includes the $index and $item variables, which represent the repeat array item's position and value, respectively.
	Resource           string                        `yaml:"resource,omitempty"             json:"resource,omitempty"`             // The URL that specifies the protocol and resource to retrieve.
	SkipInheritHeaders bool                          `yaml:"skip_inherit_headers,omitempty" json:"skip_inherit_headers,omitempty"` // Do not passthrough the headers that were sent to the template from the client's browser, even if Passthrough mode is enabled.
	Timeout            any                           `yaml:"timeout,omitempty"              json:"timeout,omitempty"`              // A duration specifying the timeout for the request.
	Transform          string                        `yaml:"transform,omitempty"            json:"transform,omitempty"`            // Specifies a JSONPath expression that can be used to transform the response data received from the binding into the data that is provided to the template.
	TlsCertificate     string                        `yaml:"tlscrt,omitempty"               json:"tlscrt,omitempty"`               // Provide the path to a TLS client certificate to present if the server requests one.
	TlsKey             string                        `yaml:"tlskey,omitempty"               json:"tlskey,omitempty"`               // Provide the path to a TLS client certificate key to present if the server requests one.
	OnlyPaths          []string                      `yaml:"only,omitempty"                 json:"only,omitempty"`                 // A list of request paths and glob patterns, ANY of which the binding will evaluate on.
	ExceptPaths        []string                      `yaml:"except,omitempty"               json:"except,omitempty"`               // A list of request paths and glob patterns, ANY of which the binding will NOT evaluate on.
	Interval           string                        `yaml:"interval,omitempty"             json:"interval,omitempty"`             // For Async Bindings, this specifies the interval on which data sources should be refreshed (if so desired).
	Restrict           any                           `yaml:"restrict,omitempty"             json:"restrict,omitempty"`             // DEPRECATED: use OnlyPaths/ExceptPaths instead.
	server             *Server
	lastRefreshedAt    time.Time
	syncing            bool
}

func (binding *Binding) shouldEvaluate(req *http.Request, data map[string]any, funcs FuncMap) error {
	if httputil.RequestGetValue(req, `force`).Bool() {
		return nil
	}

	var id = reqid(req)

	if !binding.NoTemplate {
		var proceed bool
		var desc string

		// if any inclusions are present, then ONLY a matching path will proceed
		if len(binding.OnlyPaths) > 0 {
			for _, pattern := range binding.OnlyPaths {
				if ok, err := filepath.Match(pattern, req.URL.Path); err == nil {
					if ok {
						proceed = true
						desc = fmt.Sprintf(" pattern only=%q", pattern)
						break
					}
				} else {
					return fmt.Errorf("bad 'only' pattern %q: %v", pattern, err)
				}
			}
		} else {
			// otherwise, proceed by default
			proceed = true
		}

		// if any exclusions are present, then any matching one can stop evaluation
		for _, pattern := range binding.ExceptPaths {
			if ok, err := filepath.Match(pattern, req.URL.Path); err == nil {
				if ok {
					proceed = false
					desc = fmt.Sprintf(" pattern except=%q", pattern)
					break
				}
			} else {
				return fmt.Errorf("bad 'except' pattern %q: %v", pattern, err)
			}
		}

		if !proceed {
			binding.Optional = true
			log.Debugf("[%s] Binding %q not being evaluated: path %q matched%s", id, binding.Name, req.URL.Path, desc)
			return ErrSkipEval
		}

		if binding.OnlyIfExpr != `` {
			if v, err := EvalInline(binding.OnlyIfExpr, data, funcs); err == nil {
				if !typeutil.Bool(v) {
					binding.Optional = true
					log.Debugf("[%s] Binding %q not being evaluated because only_if expression was false", id, binding.Name)
					return ErrSkipEval
				}
			} else {
				return fmt.Errorf("only_if: %v", err)
			}
		}

		if binding.NotIfExpr != `` {
			if v, err := EvalInline(binding.NotIfExpr, data, funcs); err == nil {
				if typeutil.Bool(v) {
					binding.Optional = true
					log.Debugf("[%s] Binding %q not being evaluated because not_if expression was truthy", id, binding.Name)
					return ErrSkipEval
				}
			} else {
				return fmt.Errorf("not_if: %v", err)
			}
		}
	}

	return nil
}

func (binding *Binding) tracedEvaluate(req *http.Request, header *TemplateHeader, data map[string]any, funcs FuncMap) (out any, err error) {
	var tracer opentracing.Tracer
	var spanopts []opentracing.StartSpanOption

	// if the originating request has a tracing span, we're going to create a child span of that
	// to trace this binding evaluation
	if parentSpan, ok := httputil.RequestGetValue(req, JaegerSpanKey).Value.(opentracing.Span); ok {
		tracer = binding.server.opentrace
		spanopts = append(spanopts, opentracing.ChildOf(parentSpan.Context()))
		spanopts = append(spanopts, opentracing.Tag{
			Key:   `diecast.binding`,
			Value: binding.Name,
		})
	} else {
		tracer = new(opentracing.NoopTracer)
	}

	var childSpan opentracing.Span

	if traceName := binding.server.traceName(fmt.Sprintf("Binding: %s", binding.Name)); traceName != `` {
		var traceHeaders = make(http.Header)

		childSpan = tracer.StartSpan(traceName, spanopts...)
		childSpan.Tracer().Inject(
			childSpan.Context(),
			opentracing.HTTPHeaders,
			opentracing.HTTPHeadersCarrier(traceHeaders),
		)

		if header != nil {
			if len(header.additionalHeaders) == 0 {
				header.additionalHeaders = make(map[string]any)
			}

			for k, vv := range traceHeaders {
				for _, v := range vv {
					header.additionalHeaders[k] = v
				}
			}
		}
	}

	out, err = binding.Evaluate(req, header, data, funcs)

	if childSpan != nil {
		if err != nil {
			childSpan.SetTag(`error`, err.Error())
		}

		childSpan.Finish()
	}

	return
}

func (binding *Binding) Evaluate(req *http.Request, header *TemplateHeader, data map[string]any, funcs FuncMap) (any, error) {
	var id = reqid(req)
	log.Debugf("[%s] Evaluating binding %q", id, binding.Name)

	if req.Header.Get(`X-Diecast-Binding`) == binding.Name {
		httputil.RequestSetValue(req, ContextStatusKey, http.StatusLoopDetected)
		return nil, fmt.Errorf("loop detected")
	}

	var method = strings.ToUpper(binding.Method)

	if method == `` {
		method = http.MethodGet
	}

	method = strings.ToUpper(method)
	var uri string

	if u, err := EvalInline(binding.Resource, data, funcs); err == nil {
		uri = u
	} else {
		return nil, fmt.Errorf("resource: %v", err)
	}

	// bindings may specify that a request should be made to the currently server address by
	// prefixing the URL path with a colon (":") or slash ("/").
	//
	if strings.HasPrefix(uri, `:`) || strings.HasPrefix(uri, `/`) {
		var prefix = binding.server.bestInternalLoopbackUrl(req)

		prefix = strings.TrimSuffix(prefix, `/`)
		uri = strings.TrimPrefix(uri, `:`)
		uri = strings.TrimPrefix(uri, `/`)

		uri = fmt.Sprintf("%s/%s", prefix, uri)

		// allows bindings referencing the local server to avoid TLS cert verification
		// because the host is often `localhost:port`, which probably won't verify anyway.
		if AllowInsecureLoopbackBindings {
			if bpu, err := url.Parse(uri); err == nil {
				// lookup the hostname of the requested URL. if and only if ALL of the
				// returned addresses are loopback addresses does Insecure remain true.
				if addrs, err := net.LookupIP(bpu.Hostname()); err == nil {
					binding.Insecure = true

					for _, addr := range addrs {
						if !addr.IsLoopback() {
							binding.Insecure = false
							break
						}
					}
				}
			}
		}
	}

	if err := binding.shouldEvaluate(req, data, funcs); err != nil {
		return nil, err
	}

	if reqUrl, err := url.Parse(uri); err == nil {
		reqUrl.Scheme = strings.ToLower(reqUrl.Scheme)

		var protocol Protocol

		if p, ok := registeredProtocols[reqUrl.Scheme]; ok && p != nil {
			protocol = p
		} else {
			return nil, fmt.Errorf("cannot evaluate binding %v: invalid protocol scheme %q", binding.Name, reqUrl.Scheme)
		}

		log.Debugf("[%s]  binding %q: protocol=%T uri=%v", id, binding.Name, protocol, uri)
		log.Infof("[%s] Binding: > %s %+v ? %s", id, strings.ToUpper(sliceutil.OrString(method, `get`)), reqUrl.String(), reqUrl.RawQuery)

		var additionalHeaders map[string]any

		if header != nil {
			additionalHeaders = header.additionalHeaders
		}

		if response, err := protocol.Retrieve(&ProtocolRequest{
			Verb:              method,
			URL:               reqUrl,
			Binding:           binding,
			Request:           req,
			Header:            header,
			TemplateData:      data,
			TemplateFuncs:     funcs,
			DefaultTimeout:    binding.server.bindingTimeout(),
			AdditionalHeaders: additionalHeaders,
		}); err == nil {
			defer response.Close()

			var onError BindingErrorAction

			if oe, err := EvalInline(string(binding.OnError), data, funcs); err == nil {
				onError = BindingErrorAction(oe)
			} else {
				return nil, fmt.Errorf("on_error: %v", err)
			}

			// handle per-http-status response handlers
			if len(binding.IfStatus) > 0 && response.StatusCode > 0 {
				var statusAction BindingErrorAction
				var nxx = typeutil.String(response.StatusCode - (response.StatusCode % 100))
				nxx = strings.Replace(nxx, `0`, `x`, -1)
				var nXX = strings.Replace(nxx, `0`, `X`, -1)

				// get the action for this code
				if sa, ok := binding.IfStatus[typeutil.String(response.StatusCode)]; ok && sa != `` {
					statusAction = sa
				} else if sa, ok := binding.IfStatus[nxx]; ok && sa != `` {
					statusAction = sa
				} else if sa, ok := binding.IfStatus[nXX]; ok && sa != `` {
					statusAction = sa
				} else if sa, ok := binding.IfStatus[`*`]; ok && sa != `` {
					statusAction = sa
				}

				if statusAction != `` {
					if se, err := EvalInline(string(statusAction), data, funcs); err == nil {
						statusAction = BindingErrorAction(se)
					} else {
						return nil, fmt.Errorf("if_status: %v", err)
					}

					switch statusAction {
					case ActionIgnore:
						onError = ActionIgnore
					default:
						var redirect = string(statusAction)

						if !binding.NoTemplate {
							if r, err := EvalInline(redirect, data, funcs); err == nil {
								redirect = r
							} else {
								return nil, fmt.Errorf("redirect: %v", err)
							}
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

			var data, err = io.ReadAll(response)

			if response.StatusCode >= 400 {
				err = fmt.Errorf("%v", data)
			}

			if err != nil {
				switch onError {
				case ActionPrint:
					return nil, fmt.Errorf("%v", err)
				case ActionIgnore:
					break
				default:
					var redirect = string(onError)

					// if a url or path was specified, redirect the parent request to it
					if strings.HasPrefix(redirect, `http`) || strings.HasPrefix(redirect, `/`) {
						return nil, RedirectTo(redirect)
					} else if log.ErrHasPrefix(err, `[`) {
						return nil, err
					} else {
						httputil.RequestSetValue(req, ContextStatusKey, response.StatusCode)
						return nil, fmt.Errorf("[%s] %s %v: %v", id, method, reqUrl, err)
					}
				}
			}

			if err == nil {
				mimeType, _, _ := mime.ParseMediaType(response.MimeType)

				if mimeType == `` {
					mimeType, _ = stringutil.SplitPair(response.MimeType, `;`)
				}

				// only do response body processing if there is data to process
				if len(data) > 0 {
					if binding.Parser == `` {
						switch mimeType {
						case `application/json`:
							binding.Parser = `json`
						case `application/x-yaml`, `application/yaml`, `text/yaml`:
							binding.Parser = `yaml`
						case `text/html`:
							binding.Parser = `html`
						case `text/xml`:
							binding.Parser = `xml`
						case `text/plain`:
							binding.Parser = `text`
						case `application/octet-stream`:
							binding.Parser = `literal`
						}
					}

					var rv any

					switch binding.Parser {
					case `json`, ``:
						// if the parser is unset, and the response type is NOT application/json, then
						// just read the response as plain text and return it.
						//
						// If you're certain the response actually is JSON, then explicitly set Parser==`json`
						//
						if binding.Parser == `` && mimeType != `application/json` {
							rv = string(data)
						} else {
							err = json.Unmarshal(data, &rv)
						}

					case `yaml`:
						err = yaml.UnmarshalStrict(data, &rv)

					case `html`:
						rv, err = goquery.NewDocumentFromReader(bytes.NewBuffer(data))

					case `tsv`:
						rv, err = xsvToArray(data, '\t')

					case `csv`:
						rv, err = xsvToArray(data, ',')

					case `xml`:
						rv, err = xmlToMap(data)

					case `text`:
						rv = string(data)

					case `raw`:
						rv, err = template.HTML(string(data)), nil

					case `literal`:
						rv = data

					default:
						return nil, fmt.Errorf("[%s] Unknown response parser %q", id, binding.Parser)
					}

					if err != nil {
						return nil, err
					}

					if binding.server.EnableDebugging {
						if typeutil.IsArray(rv) || typeutil.IsMap(rv) {
							if debugBody, err := json.MarshalIndent(rv, ``, `  `); err == nil {
								for _, line := range stringutil.SplitLines(debugBody, "\n") {
									log.Debugf("[%s]  [B] %s", id, line)
								}
							}
						}
					}

					return ApplyJPath(rv, binding.Transform)
				} else {
					return nil, nil
				}
			} else {
				return nil, fmt.Errorf("[%s] unhandled binding error: %v", id, err)
			}
		} else {
			return nil, fmt.Errorf("[%s] HTTP %v", id, err)
		}
	} else {
		return nil, fmt.Errorf("[%s] url: %v", id, err)
	}
}

func (binding *Binding) asyncEval() (any, error) {
	if s := binding.server; s != nil {
		if req, err := http.NewRequest(
			http.MethodGet,
			s.bestInternalLoopbackUrl(nil),
			nil,
		); err == nil {
			// informs shouldEvaluate() that we should, indeed, evaluate this one.
			httputil.RequestSetValue(req, `force`, true)

			var data = make(map[string]any)
			var funcs = s.GetTemplateFunctions(data, s.BaseHeader)

			return binding.tracedEvaluate(req, s.BaseHeader, data, funcs)
		} else {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("no server instance")
	}
}

func EvalInline(input string, data map[string]any, funcs FuncMap, names ...string) (string, error) {
	var suffix = strings.Join(names, `-`)

	if suffix != `` {
		suffix = `:` + suffix
	}

	var tmpl = NewTemplate(`inline`+suffix, TextEngine)

	if funcs == nil {
		funcs = GetStandardFunctions(nil)
	}

	tmpl.Funcs(funcs)

	if err := tmpl.ParseString(input); err == nil {
		var output = bytes.NewBuffer(nil)

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

func ShouldEvalInline(input any, data map[string]any, funcs FuncMap) typeutil.Variant {
	if ins := typeutil.String(input); strings.Contains(ins, `{{`) && strings.Contains(ins, `}}`) {
		if out, err := EvalInline(ins, data, funcs); err == nil {
			return typeutil.V(out)
		}
	}

	return typeutil.V(input)
}

func ApplyJPath(data any, jpath string) (any, error) {
	if typeutil.IsMap(data) && jpath != `` {
		var err error

		for i, line := range strings.Split(jpath, "\n") {
			line = strings.TrimSpace(line)

			if line == `` {
				continue
			}

			data, err = jsonpath.JsonPathLookup(data, line)

			if err != nil {
				return data, fmt.Errorf("jpath line %d: %v", i+1, err)
			}
		}
	}

	return data, nil
}
