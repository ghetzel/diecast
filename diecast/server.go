package diecast

import (
    "fmt"
    "io/ioutil"
    "net/http"
    "net/url"
    "os"
    "path"
    "path/filepath"
    "regexp"
    "strings"
    "github.com/julienschmidt/httprouter"
    "github.com/codegangsta/negroni"
    "github.com/shutterstock/go-stockutil/stringutil"

    "github.com/ghetzel/diecast/diecast/util"
    "github.com/ghetzel/diecast/diecast/engines"
    "github.com/ghetzel/diecast/diecast/engines/pongo"

    log "github.com/Sirupsen/logrus"
)

const DEFAULT_CONFIG_PATH   = `diecast.yml`
const DEFAULT_STATIC_PATH   = `public`
const DEFAULT_SERVE_ADDRESS = `127.0.0.1`
const DEFAULT_SERVE_PORT    = 28419

type Server struct {
    Address      string
    Port         int
    Bindings     map[string]Binding
    ConfigPath   string
    TemplatePath string
    Templates    map[string]engines.ITemplate
    StaticPath   string
    LogLevel     string

    router       *httprouter.Router
    server       *negroni.Negroni
}

func NewServer() *Server {
    return &Server{
        Address:      DEFAULT_SERVE_ADDRESS,
        Port:         DEFAULT_SERVE_PORT,
        ConfigPath:   DEFAULT_CONFIG_PATH,
        TemplatePath: engines.DEFAULT_TEMPLATE_PATH,
        Bindings:     make(map[string]Binding),
        Templates:    make(map[string]engines.ITemplate),
        StaticPath:   DEFAULT_STATIC_PATH,
    }
}

func (self *Server) Initialize() error {
    if self.LogLevel != `` {
        util.ParseLogLevel(self.LogLevel)
    }

    if data, err := ioutil.ReadFile(self.ConfigPath); err == nil {
        if config, err := LoadConfig(data); err == nil {
            if err := self.PopulateBindings(config.Bindings); err != nil {
                return fmt.Errorf("Cannot populate bindings: %v", err)
            }

        }else{
            return fmt.Errorf("Cannot load bindings.yml: %v", err)
        }
    }

    if err := self.LoadTemplates(); err != nil {
        return err
    }

    return nil
}

func (self *Server) LoadTemplates() error {
    return filepath.Walk(self.TemplatePath, func(filename string, info os.FileInfo, err error) error {
        log.Debugf("File in template path: %s (err: %v)", filename, err)

        if err == nil {
            if info.Mode().IsRegular() && !strings.HasPrefix(path.Base(filename), `_`) {
                ext := path.Ext(filename)
                key := strings.TrimSuffix(strings.TrimPrefix(filename, path.Clean(self.TemplatePath)+`/`), ext)

                if _, ok := self.Templates[key]; !ok {
                    var tpl engines.ITemplate

                    switch ext {
                    case `.pongo`:
                        tpl = pongo.New()
                    default:
                        return nil
                    }

                    tpl.SetTemplateDir(self.TemplatePath)

                    log.Debugf("Load template at %s: %T: [%s] %s", filename, tpl, key, tpl.GetTemplateDir())

                    if err := tpl.Load(key); err == nil {
                        self.Templates[key] = tpl
                    }else{
                        log.Warnf("Error loading template '%s': %v", filename, err)
                        return nil
                    }
                }else{
                    log.Warnf("Cannot load template '%s', key was already loaded", filename)
                }
            }
        }

        return nil
    })
}

func (self *Server) Serve() error {
    self.router = httprouter.New()

    self.router.GET(`/*path`, func(w http.ResponseWriter, req *http.Request, params httprouter.Params){
        routePath  := params.ByName(`path`)
        tplKey     := routePath
        var tpl engines.ITemplate

        if tplKey == `/` {
            tplKey = `/index`
        }

        parts := strings.Split(tplKey, `/`)
        parts  = parts[1:len(parts)]

        for i, _ := range parts {
            key := strings.Join(parts[0:len(parts) - i], `/`)
            // log.Infof("Trying: %s", key)

            if t, ok := self.Templates[key + `/index`]; ok {
                tpl = t
                break
            }else if t, ok := self.Templates[key]; ok {
                tpl = t
                break
            }
        }

    //  template was not found, attempt to load the index template
        if tpl == nil {
            if t, ok := self.Templates[`index`]; ok {
                tpl = t
            }
        }

        if tpl != nil {
            routeBindings    := self.GetBindings(req.Method, routePath, req)
            allParams        := make(map[string]interface{})
            payload          := map[string]interface{}{
                `route`:  params.ByName(`path`),
                `params`: allParams,
            }

            for _, binding := range routeBindings {
                for k, v := range binding.ResourceParams {
                    allParams[k] = v
                }
            }

            bindingData := make(map[string]interface{})

            for key, binding := range routeBindings {
                if data, err := binding.Evaluate(req, params); err == nil {
                    bindingData[key] = data
                }else{
                    log.Errorf("Binding '%s' failed to evaluate: %v", key, err)
                }
            }


            payload[`data`] = bindingData

            log.Debugf("Data for %s\n---\n%+v\n---\n", routePath, payload)

            if err := tpl.Render(w, payload); err != nil {
                http.Error(w, err.Error(), http.StatusInternalServerError)
            }

        }else{
            http.Error(w, fmt.Sprintf("Template '%s' not found", tplKey), http.StatusNotFound)
        }
    })

    self.server = negroni.New()
    self.server.Use(negroni.NewRecovery())
    self.server.Use(negroni.NewStatic( http.Dir(self.StaticPath) ))
    self.server.UseHandler(self.router)

    self.server.Run(fmt.Sprintf("%s:%d", self.Address, self.Port))
    return nil
}

func (self *Server) GetBindings(method string, routePath string, req *http.Request) map[string]Binding {
    var httpMethod HttpMethod
    bindings := make(map[string]Binding)


    for key, binding := range self.Bindings {
        if binding.RouteMethods == MethodAny || binding.RouteMethods & httpMethod == httpMethod {
            for _, rx := range binding.Routes {
                if match := rx.FindStringSubmatch(routePath); match != nil {
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

                            binding.Resource = &newUrl
                        }
                    }

                    bindings[key] = binding
                    break
                }
            }
        }else{
            log.Warnf("Binding '%s' did not match %s %s", key, method, routePath)
        }
    }

    // log.Debugf("Bindings for %s %s -> %v", method, routePath, bindings)
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
        self.Bindings[name] = binding
    }

    return nil
}