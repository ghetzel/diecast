package diecast

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"html"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/ghetzel/go-stockutil/httputil"
	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/typeutil"
	"github.com/gregjones/httpcache"
)

type BindingErrorAction string

const (
	ActionSummarize BindingErrorAction = `summarize`
	ActionPrint                        = `print`
	ActionContinue                     = `continue`
	ActionBreak                        = `break`
	ActionIgnore                       = `ignore`
)

var BindingClient = http.DefaultClient
var AllowInsecureLoopbackBindings bool
var DefaultParamJoiner = `;`

type Binding struct {
	Name               string                     `json:"name"`
	Restrict           []string                   `json:"restrict"`
	OnlyIfExpr         string                     `json:"only_if"`
	NotIfExpr          string                     `json:"not_if"`
	Method             string                     `json:"method"`
	Resource           string                     `json:"resource"`
	Insecure           bool                       `json:"insecure"`
	ParamJoiner        string                     `json:"param_joiner"`
	Params             map[string]interface{}     `json:"params"`
	Headers            map[string]string          `json:"headers"`
	BodyParams         map[string]interface{}     `json:"body"`
	RawBody            string                     `json:"rawbody"`
	Formatter          string                     `json:"formatter"`
	Parser             string                     `json:"parser"`
	NoTemplate         bool                       `json:"no_template"`
	Optional           bool                       `json:"optional"`
	Fallback           interface{}                `json:"fallback"`
	OnError            BindingErrorAction         `json:"on_error"`
	IfStatus           map[int]BindingErrorAction `json:"if_status"`
	Repeat             string                     `json:"repeat"`
	SkipInheritHeaders bool                       `json:"skip_inherit_headers"`
	DisableCache       bool                       `json:"disable_cache"`
	server             *Server
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

	log.Debugf("Evaluating binding %q", self.Name)

	if req.Header.Get(`X-Diecast-Binding`) == self.Name {
		return nil, fmt.Errorf("Loop detected")
	}

	method := strings.ToUpper(self.Method)

	resource := EvalInline(self.Resource, data, funcs)

	// bindings may specify that a request should be made to the currently server address by
	// prefixing the URL path with a colon (":") or slash ("/").
	//
	if strings.HasPrefix(resource, `:`) || strings.HasPrefix(resource, `/`) {
		var prefix string

		if self.server.BindingPrefix != `` {
			prefix = self.server.BindingPrefix
		} else {
			prefix = fmt.Sprintf("http://%s", req.Host)
		}

		prefix = strings.TrimSuffix(prefix, `/`)
		resource = strings.TrimPrefix(resource, `:`)
		resource = strings.TrimPrefix(resource, `/`)

		resource = fmt.Sprintf("%s/%s", prefix, resource)

		// allows bindings referencing the local server to avoid TLS cert verification
		// because the prefix is often `localhost:port`, which probably won't verify anyway.
		if AllowInsecureLoopbackBindings {
			self.Insecure = true
		}
	}

	if !self.NoTemplate {
		if self.OnlyIfExpr != `` {
			if v := EvalInline(self.OnlyIfExpr, data, funcs); typeutil.IsEmpty(v) || stringutil.IsBooleanFalse(v) {
				self.Optional = true
				return nil, fmt.Errorf("Binding %q not being evaluated because only_if expression was false", self.Name)
			}
		}

		if self.NotIfExpr != `` {
			if v := EvalInline(self.NotIfExpr, data, funcs); !typeutil.IsEmpty(v) && !stringutil.IsBooleanFalse(v) {
				self.Optional = true
				return nil, fmt.Errorf("Binding %q not being evaluated because not_if expression was truthy", self.Name)
			}
		}
	}

	log.Debugf("  binding %q: resource=%v", self.Name, resource)

	if reqUrl, err := url.Parse(resource); err == nil {
		if bindingReq, err := http.NewRequest(method, reqUrl.String(), nil); err == nil {

			// build request querystring
			// -------------------------------------------------------------------------------------

			// eval and add query string parameters to request
			qs := bindingReq.URL.Query()
			for k, v := range self.Params {
				var vS string

				if typeutil.IsArray(v) {
					joiner := DefaultParamJoiner

					if j := self.ParamJoiner; j != `` {
						joiner = j
					}

					vS = strings.Join(sliceutil.Stringify(v), joiner)
				} else {
					vS = stringutil.MustString(v)
				}

				if !self.NoTemplate {
					vS = EvalInline(vS, data, funcs)
				}

				log.Debugf("  binding %q: param %v=%v", self.Name, k, vS)
				qs.Set(k, vS)
			}

			bindingReq.URL.RawQuery = qs.Encode()

			// build request body
			// -------------------------------------------------------------------------------------
			// binding body content can be specified either as key-value pairs encoded using a
			// set of pre-defined encoders, or as a raw string (Content-Type can be explicitly set
			// via Headers).
			//
			var body bytes.Buffer

			if self.BodyParams != nil {
				bodyParams := make(map[string]interface{})

				if len(self.BodyParams) > 0 {
					// evaluate each body param value as a template (unless explicitly told not to)
					if err := maputil.Walk(self.BodyParams, func(value interface{}, path []string, isLeaf bool) error {
						if isLeaf {
							if !self.NoTemplate {
								value = EvalInline(fmt.Sprintf("%v", value), data, funcs)
							}

							maputil.DeepSet(bodyParams, path, stringutil.Autotype(value))
						}

						return nil
					}); err == nil {
						log.Debugf("  binding %q: bodyparam %#v", self.Name, bodyParams)
					} else {
						return nil, err
					}
				}

				// perform encoding of body data
				if len(bodyParams) > 0 {
					switch self.Formatter {
					case `json`, ``:
						// JSON-encode params into the body buffer
						if err := json.NewEncoder(&body).Encode(&bodyParams); err != nil {
							return nil, err
						}

						// set body and content type
						bindingReq.Body = ioutil.NopCloser(&body)
						bindingReq.Header.Set(`Content-Type`, `application/json`)

					case `form`:
						form := url.Values{}

						// add params to form values
						for k, v := range bodyParams {
							form.Add(k, fmt.Sprintf("%v", v))
						}

						// write encoded form values to body buffer
						if _, err := body.WriteString(form.Encode()); err != nil {
							return nil, err
						}

						// set body and content type
						bindingReq.Body = ioutil.NopCloser(&body)
						bindingReq.Header.Set(`Content-Type`, `application/x-www-form-urlencoded`)

					default:
						return nil, fmt.Errorf("Unknown request formatter %q", self.Formatter)
					}
				}
			} else if self.RawBody != `` {
				payload := EvalInline(self.RawBody, data, funcs)
				log.Debugf("  binding %q: rawbody %s", self.Name, payload)

				bindingReq.Body = ioutil.NopCloser(bytes.NewBufferString(payload))
			}

			// build request headers
			// -------------------------------------------------------------------------------------

			// if specified, have the binding request inherit the headers from the initiating request
			if !self.SkipInheritHeaders {
				for k, _ := range req.Header {
					v := req.Header.Get(k)
					log.Debugf("  binding %q: inherit %v=%v", self.Name, k, v)
					bindingReq.Header.Set(k, v)
				}
			}

			// add headers to request
			for k, v := range self.Headers {
				if !self.NoTemplate {
					v = EvalInline(v, data, funcs)
				}

				log.Debugf("  binding %q:  header %v=%v", self.Name, k, v)
				bindingReq.Header.Set(k, v)
			}

			bindingReq.Header.Set(`X-Diecast-Binding`, self.Name)

			log.Infof("Binding: > %s %+v ? %s", strings.ToUpper(sliceutil.OrString(method, `get`)), reqUrl.String(), reqUrl.RawQuery)

			// configure a transport with the requested SSL settings
			transport := &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: self.Insecure,
				},
			}

			// setup caching transport (if desired)
			if self.DisableCache {
				BindingClient.Transport = transport
			} else {
				BindingClient.Transport = &httpcache.Transport{
					Transport:           transport,
					Cache:               self.server.cache,
					MarkCachedResponses: true,
				}
			}

			if bindingReq.URL.Scheme == `https` && self.Insecure {
				log.Noticef("SSL/TLS certificate validation is disabled for this request.")
				log.Noticef("This is insecure as the response can be tampered with.")
			}

			// tell the server we want to close the connection when done
			bindingReq.Close = true

			// perform binding request
			// -------------------------------------------------------------------------------------
			if res, err := BindingClient.Do(bindingReq); err == nil {
				defer res.Body.Close()

				log.Infof("Binding: < HTTP %d (body: %d bytes)", res.StatusCode, res.ContentLength)

				// debug log response headers
				for k, v := range res.Header {
					log.Debugf("  [H] %v: %v", k, strings.Join(v, ` `))
				}

				onError := self.OnError

				// handle per-http-status response handlers
				if len(self.IfStatus) > 0 {
					// get the action for this code
					if statusAction, ok := self.IfStatus[res.StatusCode]; ok {
						switch statusAction {
						case ActionIgnore:
							onError = ActionIgnore
						default:
							redirect := string(statusAction)

							if !self.NoTemplate {
								redirect = EvalInline(redirect, data, funcs)
							}

							// if a url or path was specified, redirect the parent request to it
							if strings.HasPrefix(redirect, `http`) || strings.HasPrefix(redirect, `/`) {
								return nil, RedirectTo(redirect)
							} else {
								return nil, fmt.Errorf("Invalid status action '%v'", redirect)
							}
						}
					}
				}

				var reader io.Reader
				defer res.Body.Close()

				if body, err := httputil.DecodeResponse(res); err == nil {
					if closer, ok := body.(io.ReadCloser); ok {
						reader = closer
					} else {
						reader = ioutil.NopCloser(body)
					}
				}

				if data, err := ioutil.ReadAll(reader); err == nil {
					if res.StatusCode >= 400 {
						switch onError {
						case ActionPrint:
							return nil, fmt.Errorf("%v", string(data[:]))
						case ActionIgnore:
							break
						default:
							redirect := string(onError)

							// if a url or path was specified, redirect the parent request to it
							if strings.HasPrefix(redirect, `http`) || strings.HasPrefix(redirect, `/`) {
								return nil, RedirectTo(redirect)
							} else {
								return nil, fmt.Errorf(
									"Request %s %v failed: %s",
									bindingReq.Method,
									bindingReq.URL,
									res.Status,
								)
							}
						}
					}

					// only do response body processing if there is data to process
					if len(data) > 0 {
						switch self.Parser {
						case `json`, ``:
							// if the parser is unset, and the response type is NOT application/json, then
							// just read the response as plain text and return it.
							//
							// If you're certain the response actually is JSON, then explicitly set Parser==`json`
							//
							if self.Parser == `` && res.Header.Get(`Content-Type`) != `application/json` {
								return string(data), nil
							} else {
								var rv interface{}

								if err := json.Unmarshal(data, &rv); err == nil {
									return rv, nil
								} else {
									return nil, err
								}
							}

						case `html`:
							return goquery.NewDocumentFromReader(bytes.NewBuffer(data))

						case `text`:
							return string(data), nil

						case `raw`:
							return template.HTML(string(data)), nil

						default:
							return nil, fmt.Errorf("Unknown response parser %q", self.Parser)
						}
					} else {
						return nil, nil
					}
				} else {
					return nil, fmt.Errorf("Failed to read response body: %v", err)
				}
			} else {
				return nil, err
			}
		} else {
			return nil, err
		}
	} else {
		return nil, err
	}
}

func EvalInline(input string, data map[string]interface{}, funcs FuncMap) string {
	tmpl := NewTemplate(`inline`, HtmlEngine)
	tmpl.Funcs(funcs)

	if err := tmpl.Parse(input); err == nil {
		output := bytes.NewBuffer(nil)

		if err := tmpl.Render(output, data, ``); err == nil {
			// since this data may have been entity escaped by html/template, unescape it here
			return html.UnescapeString(output.String())
		} else {
			panic(fmt.Sprintf("error evaluating %q: %v", input, err))
		}
	}

	return input
}
