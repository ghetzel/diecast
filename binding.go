package diecast

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

type Binding struct {
	Name         string                 `json:"-"`
	Restrict     []string               `json:"restrict"`
	Method       string                 `json:"method"`
	Resource     string                 `json:"resource"`
	Params       map[string]interface{} `json:"params"`
	EscapeParams bool                   `json:"escape_params"`
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

func (self *Binding) Evaluate(req *http.Request) (interface{}, error) {
	method := strings.ToUpper(self.Method)

	if reqUrl, err := url.Parse(self.Resource); err == nil {
		if bindingReq, err := http.NewRequest(method, reqUrl.String(), nil); err == nil {
			client := &http.Client{}

			log.Debugf("Binding Request: %s %+v ? %s", method, reqUrl.String(), reqUrl.RawQuery)

			if res, err := client.Do(bindingReq); err == nil {
				log.Debugf("Binding Response: HTTP %d (body: %d bytes)", res.StatusCode, res.ContentLength)

				if res.StatusCode < 400 {
					if data, err := ioutil.ReadAll(res.Body); err == nil {
						var rv interface{}

						if err := json.Unmarshal(data, &rv); err == nil {
							return rv, nil
						} else {
							return nil, err
						}
					} else {
						return nil, err
					}
				} else {
					return nil, fmt.Errorf("Request failed with HTTP %d: %s", res.StatusCode, res.Status)
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
