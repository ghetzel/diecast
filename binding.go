package diecast

import (
	"bytes"
	"compress/gzip"
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

	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/typeutil"
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

var DefaultParamJoiner = `;`

type Binding struct {
	Name               string                 `json:"name"`
	Restrict           []string               `json:"restrict"`
	OnlyIfExpr         string                 `json:"only_if"`
	NotIfExpr          string                 `json:"not_if"`
	Method             string                 `json:"method"`
	Resource           string                 `json:"resource"`
	ParamJoiner        string                 `json:"param_joiner"`
	Params             map[string]interface{} `json:"params"`
	Headers            map[string]string      `json:"headers"`
	BodyParams         map[string]interface{} `json:"body"`
	RawBody            string                 `json:"rawbody"`
	Formatter          string                 `json:"formatter"`
	Parser             string                 `json:"parser"`
	NoTemplate         bool                   `json:"no_template"`
	Optional           bool                   `json:"optional"`
	Fallback           interface{}            `json:"fallback"`
	OnError            BindingErrorAction     `json:"on_error"`
	Repeat             string                 `json:"repeat"`
	SkipInheritHeaders bool                   `json:"skip_inherit_headers"`
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

	// bindings may specify that a request should be made to the currently server address by
	// prefixing the URL path with a colon (":") or slash ("/").
	//
	if strings.HasPrefix(self.Resource, `:`) || strings.HasPrefix(self.Resource, `/`) {
		var prefix string

		if self.server.BindingPrefix != `` {
			prefix = self.server.BindingPrefix
		} else {
			prefix = fmt.Sprintf("http://%s", req.Host)
		}

		prefix = strings.TrimSuffix(prefix, `/`)
		resource := self.Resource
		resource = strings.TrimPrefix(resource, `:`)
		resource = strings.TrimPrefix(resource, `/`)

		self.Resource = fmt.Sprintf("%s/%s", prefix, resource)
	}

	if !self.NoTemplate {
		if self.OnlyIfExpr != `` {
			if v := EvalInline(self.OnlyIfExpr, data, funcs); typeutil.IsEmpty(v) {
				return nil, fmt.Errorf("Binding not being evaluated because only_if expression was false")
			}
		}

		if self.NotIfExpr != `` {
			if v := EvalInline(self.NotIfExpr, data, funcs); !typeutil.IsEmpty(v) {
				return nil, fmt.Errorf("Binding not being evaluated because not_if expression was truthy")
			}
		}

		self.Resource = EvalInline(self.Resource, data, funcs)
	}

	log.Debugf("  binding %q: resource=%v", self.Name, self.Resource)

	if reqUrl, err := url.Parse(self.Resource); err == nil {
		if bindingReq, err := http.NewRequest(method, reqUrl.String(), nil); err == nil {
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

			// add body to request
			var body bytes.Buffer

			if self.BodyParams != nil {
				bodyParams := make(map[string]interface{})

				if len(self.BodyParams) > 0 {
					if err := maputil.Walk(self.BodyParams, func(value interface{}, path []string, isLeaf bool) error {
						if isLeaf {
							if !self.NoTemplate {
								value = EvalInline(fmt.Sprintf("%v", value), data, funcs)
							}

							maputil.DeepSet(bodyParams, path, stringutil.Autotype(value))
						}

						return nil
					}); err == nil {
						log.Debugf("  binding %q: bodyparam %#+v", self.Name, bodyParams)
					} else {
						return nil, err
					}
				}

				if len(bodyParams) > 0 {
					switch self.Formatter {
					case `json`, ``:
						if err := json.NewEncoder(&body).Encode(&bodyParams); err != nil {
							return nil, err
						}

						bindingReq.Body = ioutil.NopCloser(&body)

					default:
						return nil, fmt.Errorf("Unknown request formatter %q", self.Formatter)
					}
				}
			} else if self.RawBody != `` {
				payload := EvalInline(self.RawBody, data, funcs)
				log.Debugf("  binding %q: rawbody %s", self.Name, payload)

				bindingReq.Body = ioutil.NopCloser(bytes.NewBufferString(payload))
			}

			bindingReq.Header.Set(`X-Diecast-Binding`, self.Name)

			log.Infof("Binding: > %s %+v ? %s", strings.ToUpper(sliceutil.OrString(method, `get`)), reqUrl.String(), reqUrl.RawQuery)

			if res, err := BindingClient.Do(bindingReq); err == nil {
				log.Infof("Binding: < HTTP %d (body: %d bytes)", res.StatusCode, res.ContentLength)
				for k, v := range res.Header {
					log.Debugf("  %v=%v", k, strings.Join(v, ` `))
				}

				var reader io.ReadCloser

				switch res.Header.Get(`Content-Encoding`) {
				case `gzip`:
					reader, err = gzip.NewReader(res.Body)
					defer reader.Close()
				default:
					reader = res.Body
				}

				if data, err := ioutil.ReadAll(reader); err == nil {
					if res.StatusCode < 400 {
						if res.ContentLength > 0 {
							switch self.Parser {
							case `json`, ``:
								var rv interface{}

								if err := json.Unmarshal(data, &rv); err == nil {
									return rv, nil
								} else {
									return nil, err
								}

							case `raw`:
								return template.HTML(string(data)), nil

							default:
								return nil, fmt.Errorf("Unknown response parser %q", self.Parser)
							}
						} else {
							return nil, nil
						}
					} else {
						switch self.OnError {
						case ActionPrint:
							return nil, fmt.Errorf("%v", string(data[:]))
						case ActionIgnore:
							return nil, nil
						default:
							return nil, fmt.Errorf("Request %s %v failed: %s",
								bindingReq.Method,
								bindingReq.URL,
								res.Status)
						}
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
