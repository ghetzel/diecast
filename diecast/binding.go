package diecast

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/url"
    "net/http"
    "regexp"
    "strings"
    "github.com/julienschmidt/httprouter"
    log "github.com/Sirupsen/logrus"
)

type HttpMethod int
const (
    MethodAny    HttpMethod = 0
    MethodGet               = 1
    MethodPost              = 2
    MethodPut               = 4
    MethodDelete            = 8
    MethodHead              = 16
    MethodOptions           = 32
    MethodPatch             = 64
)

type BindingConfig struct {
    Routes         []string          `json:"routes"`
    Resource       string            `json:"resource"`
    ResourceParams map[string]string `json:"params,omitempty"`
    RouteMethods   []string          `json:"route_methods,omitempty"`
    ResourceMethod string            `json:"resource_method,omitempty"`
    EscapeParams   bool              `json:"escape_params,omitempty"`
}

type Binding struct {
    Routes         []*regexp.Regexp
    RouteMethods   HttpMethod
    RouteParams    map[string]interface{}
    ResourceMethod HttpMethod
    Resource       *url.URL
    ResourceParams map[string]interface{}
    EscapeParams   bool
}

func (self *Binding) Evaluate(req *http.Request, params httprouter.Params) (interface{}, error) {
    var method string

    switch self.ResourceMethod {
    case MethodPost:
        method = `POST`
    case MethodPut:
        method = `PUT`
    case MethodDelete:
        method = `DELETE`
    default:
        method = `GET`
    }

    reqUrl := self.Resource.String()

    if bindingReq, err := http.NewRequest(method, reqUrl, nil); err == nil {
        client := &http.Client{}

        log.Debugf("Binding Request: %s %+v ? %s", method, self.Resource, self.Resource.RawQuery)

        if res, err := client.Do(bindingReq); err == nil {
            log.Debugf("Binding Response: HTTP %d (body: %d bytes)", res.StatusCode, res.ContentLength)

            if res.StatusCode < 400 {
                if data, err := ioutil.ReadAll(res.Body); err == nil {
                    var rv interface{}

                    if err := json.Unmarshal(data, &rv); err == nil {
                        return rv, nil
                    }else{
                        return nil, err
                    }
                }else{
                    return nil, err
                }
            }else{
                return nil, fmt.Errorf("Request failed with HTTP %d: %s", res.StatusCode, res.Status)
            }
        }else{
            return nil, err
        }
    }else{
        return nil, err
    }
}

func ToHttpMethod(method string) HttpMethod {
    switch strings.ToLower(method) {
    case `get`:
        return MethodGet
    case `post`:
        return MethodPost
    case `put`:
        return MethodPut
    case `delete`:
        return MethodDelete
    case `head`:
        return MethodHead
    case `options`:
        return MethodOptions
    case `patch`:
        return MethodPatch
    default:
        return MethodAny
    }
}
