package main

import (
    "fmt"
    "regexp"
    "net/http"
    "net/url"
    "strings"
    "os"
    "io/ioutil"
    "encoding/json"
    "strconv"
    "html/template"
    "github.com/yosssi/ace"
    "github.com/julienschmidt/httprouter"
    "github.com/codegangsta/negroni"
    "github.com/ghodss/yaml"
    "github.com/shutterstock/go-stockutil/stringutil"
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
}

type Config struct {
    Bindings map[string]BindingConfig `json:"bindings"`
}

type Binding struct {
    Routes         []*regexp.Regexp
    RouteMethods   HttpMethod
    ResourceMethod HttpMethod
    Resource       *url.URL
    ResourceParams map[string]interface{}
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

    if qs := self.Resource.RawQuery; qs != `` {
        reqUrl = reqUrl + `?` + qs
    }

    if bindingReq, err := http.NewRequest(method, reqUrl, nil); err == nil {
        client := &http.Client{}

        log.Warnf("REQ %s %+v ? %+v", method, self.Resource, self.Resource.Query())

        if res, err := client.Do(bindingReq); err == nil && res.StatusCode < 400 {
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
            return nil, err
        }
    }else{
        return nil, err
    }
}

var Bindings = make(map[string]Binding)
var TemplateFunctions = make(template.FuncMap)

func main() {
    if data, err := ioutil.ReadFile(`./bindings.yml`); err == nil {
        if config, err := LoadConfig(data); err == nil {
            if err := PopulateBindings(config.Bindings); err != nil {
                log.Fatalf("Cannot populate bindings: %v", err)
            }

        }else{
            log.Fatalf("Cannot load bindings.yml: %v", err)
        }
    }else{
        log.Fatalf("Cannot load bindings.yml: %v", err)
    }

    SiSuffixes := []string{ `bytes`, `KB`, `MB`, `GB`, `TB`, `PB`, `EB`, `YB` }

    TemplateFunctions[`autosize`] = func(input float64, fixTo int) (string, error) {
        check := float64(input)
        i := 1

        for i = 1; i < 9; i++ {
            if check < 1024.0 {
                break
            }else{
                check = (check / 1024.0)
            }
        }

        return (strconv.FormatFloat(check, 'f', fixTo, 64) + ` ` + SiSuffixes[i-1]), nil
    }

    TemplateFunctions[`length`] = func(set []interface{}) int {
        return len(set)
    }

    TemplateFunctions[`str`] = func(in ...interface{}) (string, error) {
        if len(in) > 0 {
            if in[0] != nil {
                return stringutil.ToString(in[0])
            }
        }

        return ``, nil
    }


    router := httprouter.New()

    router.GET(`/*path`, func(w http.ResponseWriter, req *http.Request, params httprouter.Params){
        path             := params.ByName(`path`)
        routeBindings    := GetBindings(req.Method, path, req)
        allParams        := make(map[string]interface{})

        for _, binding := range routeBindings {
            for k, v := range binding.ResourceParams {
                allParams[k] = v
            }
        }

        innerTplPath     := path

        if innerTplPath == `/` {
            innerTplPath = `/default`
        }

        parts := strings.Split(innerTplPath, `/`)
        parts  = parts[1:len(parts)]

        for i, _ := range parts {
            slug := strings.Join(parts[0:len(parts) - i], `/`)


            log.Infof("Trying: %s", fmt.Sprintf("%s/%s/index.ace", `templates`, slug))

            if _, err := os.Stat( fmt.Sprintf("%s/%s/index.ace", `templates`, slug) ); err == nil {
                innerTplPath = `/` + slug + `/index`
                break
            }else if os.IsNotExist(err) {
                log.Infof("Trying: %s", fmt.Sprintf("%s/%s.ace", `templates`, slug))

                if _, err := os.Stat( fmt.Sprintf("%s/%s.ace", `templates`, slug) ); err == nil {
                    innerTplPath = `/` + slug
                    break
                }
            }
        }


        log.Infof("PATH: %s", innerTplPath)

        if tpl, err := ace.Load(`base`, innerTplPath, &ace.Options{
            DynamicReload: true,
            BaseDir:       `templates`,
            FuncMap:       TemplateFunctions,
        }); err == nil {


            payload := map[string]interface{}{
                `TemplatePath`: params.ByName(`path`),
                `Params`:       allParams,
            }

            bindingData := make(map[string]interface{})

            for key, binding := range routeBindings {
                if data, err := binding.Evaluate(req, params); err == nil {
                    bindingData[key] = data
                }else{
                    log.Errorf("Binding '%s' failed to evaluate: %v", key, err)
                }
            }

            // log.Infof("Data for %s\n---\n%+v\n---\n", path, bindingData)

            payload[`data`] = bindingData

            if err := tpl.Execute(w, payload); err != nil {
                http.Error(w, err.Error(), http.StatusInternalServerError)
            }
        }else{
            http.Error(w, err.Error(), http.StatusInternalServerError)
        }
    })

    server := negroni.Classic()
    server.UseHandler(router)

    server.Run(fmt.Sprintf("%s:%d", ``, 8080))
}

func GetBindings(method string, path string, req *http.Request) map[string]Binding {
    var httpMethod HttpMethod
    bindings := make(map[string]Binding)


    for key, binding := range Bindings {
        if binding.RouteMethods == MethodAny || binding.RouteMethods & httpMethod == httpMethod {
            for _, rx := range binding.Routes {
                if match := rx.FindStringSubmatch(path); match != nil {
                    for i, matchGroupName := range rx.SubexpNames() {
                        if matchGroupName != `` {
                            newUrl := *binding.Resource
                            newUrl.Path = strings.Replace(newUrl.Path, `:`+matchGroupName, match[i], -1)

                            for qs, v := range newUrl.Query() {
                                qsv := strings.Replace(v[0], `:`+matchGroupName, match[i], -1)
                                binding.ResourceParams[qs] = qsv
                            }

                            for qs, v := range req.URL.Query() {
                                if len(v) > 0 {
                                    binding.ResourceParams[qs] = v[0]
                                }
                            }

                            rawQuery := make([]string, 0)

                            for k, v := range binding.ResourceParams {
                                if str, err := stringutil.ToString(v); err == nil {
                                    rawQuery = append(rawQuery, k + `=` + url.QueryEscape(str))
                                }
                            }

                            newUrl.RawQuery = strings.Join(rawQuery, `&`)
                            log.Infof("NEW URL %+v ? %+v", newUrl, newUrl.Query())

                            binding.Resource = &newUrl
                        }
                    }

                    bindings[key] = binding
                    break
                }
            }
        }else{
            log.Warnf("Binding '%s' did not match %s %s", key, method, path)
        }
    }

    log.Infof("Bindings for %s %s -> %v", method, path, bindings)
    return bindings
}

func PopulateBindings(bindings map[string]BindingConfig) error {
    for name, bindingConfig := range bindings {
        binding := Binding{
            ResourceParams: make(map[string]interface{}),
        }

        if len(bindingConfig.RouteMethods) == 0 {
            binding.RouteMethods = MethodAny
        }else{
            for _, method := range bindingConfig.RouteMethods {
                binding.RouteMethods = (binding.RouteMethods | ToHttpMethod(method))
            }
        }

        if bindingConfig.ResourceMethod == `` {
            binding.ResourceMethod = MethodGet
        }else{
            binding.ResourceMethod = ToHttpMethod(bindingConfig.ResourceMethod)
        }

        for _, rxstr := range bindingConfig.Routes {
            if rx, err := regexp.Compile(rxstr); err == nil {
                binding.Routes = append(binding.Routes, rx)
            }else{
                return err
            }
        }

        if u, err := url.Parse(bindingConfig.Resource); err == nil {
            qs := make([]string, 0)

            for param, value := range bindingConfig.ResourceParams {
                qs = append(qs, param + `=` + value)
            }

            if len(qs) > 0 {
                u.RawQuery = strings.Join(qs, `&`)
            }

            binding.Resource = u
        }else{
            return err
        }


        log.Infof("Setting up binding %s: %+v", name, binding)
        Bindings[name] = binding
    }

    return nil
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


func LoadConfig(data []byte) (Config, error) {
    rv := Config{}
    err := yaml.Unmarshal(data, &rv)
    return rv, err
}
