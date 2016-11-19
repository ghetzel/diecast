package diecast

import (
	"encoding/json"
	"fmt"
	"github.com/codegangsta/negroni"
	"github.com/ghetzel/diecast/engines"
	"github.com/julienschmidt/httprouter"
	"github.com/op/go-logging"
	"github.com/shutterstock/go-stockutil/sliceutil"
	"io/ioutil"
	"net/http"
	"strings"
)

var log = logging.MustGetLogger(`diecast`)

const DEFAULT_CONFIG_PATH = `diecast.yml`
const DEFAULT_STATIC_PATH = `static`
const DEFAULT_SERVE_ADDRESS = `127.0.0.1`
const DEFAULT_SERVE_PORT = 28419
const DEFAULT_ROUTE_PREFIX = `/`

var ParamDelimPre = `#{`
var ParamDelimPost = `}`

type Handler struct {
	Pattern string
	Handler http.Handler
}

type HandleFunc struct {
	Pattern    string
	HandleFunc func(http.ResponseWriter, *http.Request)
}

type Server struct {
	Address       string
	Port          int
	MountProxy    *MountProxy
	Config        Config
	ConfigPath    string
	DefaultEngine string
	TemplatePath  string
	StaticPath    string
	RoutePrefix   string
	Payload       map[string]interface{}
	Handlers      []Handler
	HandleFuncs   []HandleFunc
	mux           *http.ServeMux
	router        *httprouter.Router
	server        *negroni.Negroni
}

func NewServer() *Server {
	return &Server{
		Address:       DEFAULT_SERVE_ADDRESS,
		ConfigPath:    DEFAULT_CONFIG_PATH,
		DefaultEngine: engines.DEFAULT_TEMPLATE_ENGINE,
		Handlers:      make([]Handler, 0),
		HandleFuncs:   make([]HandleFunc, 0),
		MountProxy:    &MountProxy{},
		Payload:       make(map[string]interface{}),
		Port:          DEFAULT_SERVE_PORT,
		RoutePrefix:   DEFAULT_ROUTE_PREFIX,
		StaticPath:    DEFAULT_STATIC_PATH,
		TemplatePath:  engines.DEFAULT_TEMPLATE_PATH,
	}
}

func (self *Server) Initialize() error {
	if data, err := ioutil.ReadFile(self.ConfigPath); err == nil {
		if config, err := LoadConfig(data); err == nil {
			self.Config = config

			if v := self.Config.Options.DefaultEngine; v != `` {
				DefaultTemplateEngine = v
			}

			for name, binding := range self.Config.Bindings {
				if err := binding.Initialize(name); err != nil {
					return fmt.Errorf("Failed to initilize binding '%s': %v", name, err)
				}
			}

			if err := self.InitializeMounts(config.Mounts); err != nil {
				return fmt.Errorf("Failed to initialize mounts: %v", err)
			}
		} else {
			return fmt.Errorf("Cannot load configuration at %s: %v", self.ConfigPath, err)
		}
	}

	self.MountProxy.Fallback = http.Dir(self.StaticPath)

	// setup servemux and routing
	self.mux = http.NewServeMux()
	self.router = httprouter.New()

	self.RoutePrefix = strings.TrimSuffix(self.RoutePrefix, `/`)

	// load route handlers
	if err := self.LoadRoutes(); err != nil {
		return err
	}

	staticHandler := negroni.NewStatic(self.MountProxy)

	if self.RoutePrefix != DEFAULT_ROUTE_PREFIX {
		staticHandler.Prefix = self.RoutePrefix
	}

	self.mux.HandleFunc(fmt.Sprintf("%s/_diecast", self.RoutePrefix), func(w http.ResponseWriter, req *http.Request) {
		if data, err := json.Marshal(self); err == nil {
			w.Header().Set(`Content-Type`, `application/json`)

			if _, err := w.Write(data); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	self.mux.HandleFunc(fmt.Sprintf("%s/_bindings", self.RoutePrefix), func(w http.ResponseWriter, req *http.Request) {
		if data, err := json.Marshal(self.Config.Bindings); err == nil {
			w.Header().Set(`Content-Type`, `application/json`)

			if _, err := w.Write(data); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	// add custom http.Handlers to the mux
	for i, handler := range self.Handlers {
		log.Debugf("Setting up custom handler %d for pattern '%s'", i, handler.Pattern)
		self.mux.Handle(handler.Pattern, handler.Handler)
	}

	// add custom http.HandleFuncs to the mux
	for i, handleFunc := range self.HandleFuncs {
		log.Debugf("Setting up custom handler function %d for pattern '%s'", i, handleFunc.Pattern)
		self.mux.HandleFunc(handleFunc.Pattern, handleFunc.HandleFunc)
	}

	// fallback to httprouter (these are the routes defined in the configuration)
	self.mux.Handle(`/`, self.router)

	self.server = negroni.New()
	self.server.Use(negroni.NewRecovery())
	self.server.Use(staticHandler)
	self.server.Use(negroni.Wrap(self.mux))

	return nil
}

func (self *Server) SetPayload(key string, value interface{}) {
	self.Payload[key] = value
}

func (self *Server) LoadRoutes() error {
	for i, route := range self.Config.Routes {
		route.Index = i

		if err := route.Initialize(); err == nil {
			if err := route.LoadTemplate(self.TemplatePath); err == nil {
				for _, method := range route.Methods {
					self.setupTemplateHandler(method, route)
				}
			} else {
				return err
			}
		} else {
			return err
		}
	}

	return nil
}

func (self *Server) Serve() error {
	self.server.Run(fmt.Sprintf("%s:%d", self.Address, self.Port))
	return nil
}

func (self *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	self.server.ServeHTTP(w, r)
}

func (self *Server) GetBindings(req *http.Request, route *Route) map[string]Binding {
	bindings := make(map[string]Binding)

	//  for each of this route's bindings...
	for _, key := range route.Bindings {
		//  if the named binding exists...
		if binding, ok := self.Config.Bindings[key]; ok {
			bindings[key] = binding
		}
	}

	return bindings
}

func (self *Server) InitializeMounts(mountsConfig []Mount) error {
	mounts := make([]Mount, 0)

	for _, mount := range mountsConfig {
		log.Debugf("Initializing mount at %s", mount.Path)

		if err := mount.Initialize(); err != nil {
			return err
		}

		mounts = append(mounts, mount)
	}

	self.MountProxy.Mounts = mounts
	return nil
}

func (self *Server) setupTemplateHandler(method string, route *Route) {
	method = strings.ToUpper(method)
	endpointPath := route.Path

	if self.RoutePrefix != `` {
		endpointPath = fmt.Sprintf("%s/%s", self.RoutePrefix, endpointPath)
	}

	log.Debugf("Creating endpoint: %s %s", method, endpointPath)

	self.router.Handle(method, endpointPath, func(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
		allParams := make(map[string]interface{})
		bindingData := make(map[string]interface{})

		//  evaluate all bindings that this route is attached to
		routeBindings := self.GetBindings(req, route)

		//  populate data values for evaluated bindings
		for _, binding := range routeBindings {
			for k, v := range binding.Params {
				allParams[k] = v
			}
		}

		payload := map[string]interface{}{
			`route`: map[string]interface{}{
				`index`:   route.Index,
				`pattern`: route.Path,
				`path`:    endpointPath,
			},
			`request`: map[string]interface{}{
				`method`: req.Method,
				`url`: map[string]interface{}{
					`full`:   req.URL.String(),
					`scheme`: req.URL.Scheme,
					`host`:   req.URL.Host,
					`path`:   req.URL.Path,
					`query`:  req.URL.Query(),
				},
				`headers`: req.Header,
				`protocol`: map[string]interface{}{
					`name`:  req.Proto,
					`major`: req.ProtoMajor,
					`minor`: req.ProtoMinor,
				},
				`length`: req.ContentLength,
				`remote`: map[string]interface{}{
					`address`: req.RemoteAddr,
				},
			},
			`params`: allParams,
		}

		for key, binding := range routeBindings {
			if data, err := binding.Evaluate(req, params); err == nil {
				bindingData[key] = data
			} else {
				log.Errorf("Binding '%s' failed to evaluate: %v", key, err)
			}
		}

		payload[`data`] = bindingData

		protectedKeys := make([]string, 0)

		for k, _ := range payload {
			protectedKeys = append(protectedKeys, k)
		}

		//  add global config payload keys
		for k, v := range self.Config.Options.Payload {
			if !sliceutil.ContainsString(protectedKeys, k) {
				payload[k] = v
			}
		}

		//  add external payload keys
		for k, v := range self.Payload {
			if !sliceutil.ContainsString(protectedKeys, k) {
				payload[k] = v
			}
		}

		// add global config headers
		for name, value := range self.Config.Options.Headers {
			w.Header().Set(name, value)
		}

		// add route config headers
		for name, value := range route.Headers {
			w.Header().Set(name, value)
		}

		if err := route.Render(w, payload); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}
