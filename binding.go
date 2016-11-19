package diecast

import (
	"encoding/json"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

type Binding struct {
	Name         string                 `json:"-"`
	Method       string                 `json:"method"`
	Resource     string                 `json:"resource"`
	Params       map[string]interface{} `json:"params"`
	EscapeParams bool                   `json:"escape_params"`

	url *url.URL
}

func (self *Binding) Initialize(name string) error {
	self.Name = name

	log.Debugf("Initialize binding '%s'", self.Name)

	if u, err := url.Parse(self.Resource); err == nil {
		self.url = u
	} else {
		return err
	}

	return nil
}

func (self *Binding) Evaluate(req *http.Request, params httprouter.Params) (interface{}, error) {
	method := strings.ToUpper(self.Method)
	reqUrl := self.url.String()

	if bindingReq, err := http.NewRequest(method, reqUrl, nil); err == nil {
		client := &http.Client{}

		log.Debugf("Binding Request: %s %+v ? %s", method, self.url, self.url.RawQuery)

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
}
