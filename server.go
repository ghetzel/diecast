package main

import (
    "fmt"
    "regexp"
    "net/http"
    "net/url"
    "strings"
    "os"
    "github.com/yosssi/ace"
    "github.com/julienschmidt/httprouter"
    "github.com/codegangsta/negroni"
    "github.com/shutterstock/go-stockutil/stringutil"
    "github.com/ghetzel/diecast/functions"
    log "github.com/Sirupsen/logrus"
)

const DEFAULT_CONFIG_PATH = `./diecast.yml`

type Server struct {
    Address    string
    Port       int
    Bindings   map[string]Binding
    ConfigPath string

    router     *httprouter.Router
    server     *negroni.Negroni
}

func NewServer() *Server {
    return &Server{
        Address:    `127.0.0.1`,
        Port:       28419,
        ConfigPath: DEFAULT_CONFIG_PATH,
        Bindings:   make(map[string]Binding),
    }
}

func (self *Server) Initialize() error {
    if data, err := ioutil.ReadFile(self.ConfigPath); err == nil {
        if config, err := LoadConfig(data); err == nil {
            if err := self.PopulateBindings(config.Bindings); err != nil {
                return fmt.Errorf("Cannot populate bindings: %v", err)
            }

        }else{
            return fmt.Errorf("Cannot load bindings.yml: %v", err)
        }
    }else{
        return fmt.Errorf("Cannot load bindings.yml: %v", err)
    }

    return nil
}

func (self *Server) Serve() error {
    self.router = httprouter.New()

    self.router.GET(`/*path`, func(w http.ResponseWriter, req *http.Request, params httprouter.Params){
        path             := params.ByName(`path`)
        routeBindings    := self.GetBindings(req.Method, path, req)
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
            FuncMap:       functions.GetBaseFunctions(),
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

    self.server = negroni.Classic()
    self.server.UseHandler(router)

    self.server.Run(fmt.Sprintf("%s:%d", ``, 8080))
}

func (self *Server) GetBindings(method string, path string, req *http.Request) map[string]Binding {
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

func (self *Server) PopulateBindings(bindings map[string]BindingConfig) error {
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