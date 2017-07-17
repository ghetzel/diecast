package diecast

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/typeutil"
)

type BindingErrorAction string

const (
	ActionSummarize BindingErrorAction = `summarize`
	ActionPrint                        = `print`
)

var BindingClient = http.DefaultClient

type Binding struct {
	Name       string             `json:"name"`
	Restrict   []string           `json:"restrict"`
	OnlyIfExpr string             `json:"only_if"`
	NotIfExpr  string             `json:"not_if"`
	Method     string             `json:"method"`
	Resource   string             `json:"resource"`
	Params     map[string]string  `json:"params"`
	Headers    map[string]string  `json:"headers"`
	BodyParams map[string]string  `json:"body"`
	Formatter  string             `json:"formatter"`
	NoTemplate bool               `json:"no_template"`
	Optional   bool               `json:"optional"`
	OnError    BindingErrorAction `json:"on_error"`
	server     *Server
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

func (self *Binding) Evaluate(req *http.Request, header *TemplateHeader, data map[string]interface{}, funcs template.FuncMap) (interface{}, error) {
	log.Debugf("Evaluating binding %q", self.Name)

	if req.Header.Get(`X-Diecast-Binding`) == self.Name {
		return nil, fmt.Errorf("Loop detected")
	}

	method := strings.ToUpper(self.Method)

	// bindings may specify that a request should be made to the currently server address by
	// prefixing the URL path with a colon (":").
	//
	if strings.HasPrefix(self.Resource, `:`) {
		var prefix string

		if self.server.BindingPrefix != `` {
			prefix = self.server.BindingPrefix
		} else {
			prefix = fmt.Sprintf("http://%s", req.Host)
		}

		self.Resource = fmt.Sprintf("%s/%s",
			strings.TrimSuffix(prefix, `/`),
			strings.TrimPrefix(strings.TrimPrefix(self.Resource, `:`), `/`),
		)
	}

	if !self.NoTemplate {
		if self.OnlyIfExpr != `` {
			if v := self.Eval(self.OnlyIfExpr, data, funcs); typeutil.IsEmpty(v) {
				return nil, fmt.Errorf("Binding not being evaluated because only_if expression was false")
			}
		}

		if self.NotIfExpr != `` {
			if v := self.Eval(self.NotIfExpr, data, funcs); !typeutil.IsEmpty(v) {
				return nil, fmt.Errorf("Binding not being evaluated because not_if expression was truthy")
			}
		}

		self.Resource = self.Eval(self.Resource, data, funcs)
	}

	log.Debugf("  binding %q: resource=%v", self.Name, self.Resource)

	if reqUrl, err := url.Parse(self.Resource); err == nil {
		if bindingReq, err := http.NewRequest(method, reqUrl.String(), nil); err == nil {
			// eval and add query string parameters to request
			qs := bindingReq.URL.Query()

			for k, v := range self.Params {
				if !self.NoTemplate {
					v = self.Eval(v, data, funcs)
				}

				log.Debugf("  binding %q: param %v=%v", self.Name, k, v)
				qs.Set(k, v)
			}

			bindingReq.URL.RawQuery = qs.Encode()

			// add headers to request
			for k, v := range self.Headers {
				if !self.NoTemplate {
					v = self.Eval(v, data, funcs)
				}

				log.Debugf("  binding %q: header %v=%v", self.Name, k, v)
				bindingReq.Header.Set(k, v)
			}

			// add body to request
			var body bytes.Buffer

			if self.BodyParams != nil {
				bodyParams := make(map[string]interface{})

				for k, v := range self.BodyParams {
					if !self.NoTemplate {
						v = self.Eval(v, data, funcs)
					}

					log.Debugf("  binding %q: bodyparam %v=%v", self.Name, k, v)
					bodyParams[k] = stringutil.Autotype(v)
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
			}

			bindingReq.Header.Set(`X-Diecast-Binding`, self.Name)

			log.Debugf("Binding Request: %s %+v ? %s", method, reqUrl.String(), reqUrl.RawQuery)

			if res, err := BindingClient.Do(bindingReq); err == nil {
				log.Debugf("Binding Response: HTTP %d (body: %d bytes)", res.StatusCode, res.ContentLength)

				if data, err := ioutil.ReadAll(res.Body); err == nil {
					if res.StatusCode < 400 {
						var rv interface{}

						if err := json.Unmarshal(data, &rv); err == nil {
							return rv, nil
						} else {
							return nil, err
						}
					} else {
						switch self.OnError {
						case ActionPrint:
							return nil, fmt.Errorf("%v", string(data[:]))
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

func (self *Binding) Eval(input string, data map[string]interface{}, funcs template.FuncMap) string {
	tmpl := template.New(`inline`)
	tmpl.Funcs(funcs)

	if _, err := tmpl.Parse(input); err == nil {
		output := bytes.NewBuffer(nil)

		if err := tmpl.Execute(output, data); err == nil {
			return output.String()
		} else {
			log.Debugf("error evaluating %q: %v", input, err)
		}
	}

	return input
}
