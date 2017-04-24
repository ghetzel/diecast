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
	Method     string             `json:"method"`
	Resource   string             `json:"resource"`
	Params     map[string]string  `json:"params"`
	Headers    map[string]string  `json:"headers"`
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

func (self *Binding) Evaluate(req *http.Request, header *TemplateHeader) (interface{}, error) {
	log.Debugf("Evaluating binding %q", self.Name)

	if req.Header.Get(`X-Diecast-Binding`) == self.Name {
		return nil, fmt.Errorf("Loop detected")
	}

	method := strings.ToUpper(self.Method)
	var evalData map[string]interface{}

	if !self.NoTemplate {
		evalData = requestToEvalData(req, header)
	}

	if strings.HasPrefix(self.Resource, `:`) {
		self.Resource = fmt.Sprintf("http://%s/%s",
			req.Host,
			strings.TrimPrefix(strings.TrimPrefix(self.Resource, `:`), `/`))
	}

	if !self.NoTemplate {
		self.Resource = self.Eval(self.Resource, evalData)
	}

	if reqUrl, err := url.Parse(self.Resource); err == nil {
		if bindingReq, err := http.NewRequest(method, reqUrl.String(), nil); err == nil {
			// eval and add query string parameters to request
			qs := bindingReq.URL.Query()

			for k, v := range self.Params {
				if !self.NoTemplate {
					v = self.Eval(v, evalData)
				}

				qs.Set(k, v)
			}

			bindingReq.URL.RawQuery = qs.Encode()

			// add headers to request
			for k, v := range self.Headers {
				if !self.NoTemplate {
					v = self.Eval(v, evalData)
				}

				bindingReq.Header.Set(k, v)
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

func (self *Binding) Eval(input string, data map[string]interface{}) string {
	tmpl := template.New(`inline`)
	tmpl.Funcs(GetStandardFunctions())

	if self.server != nil && self.server.AdditionalFunctions != nil {
		tmpl.Funcs(self.server.AdditionalFunctions)
	}

	if _, err := tmpl.Parse(input); err == nil {
		output := bytes.NewBuffer(nil)

		if err := tmpl.Execute(output, data); err == nil {
			return output.String()
		}
	}

	return input
}
