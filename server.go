package diecast

import (
	"encoding/json"
	"fmt"
	"github.com/codegangsta/negroni"
	"github.com/julienschmidt/httprouter"
	"github.com/op/go-logging"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var log = logging.MustGetLogger(`diecast`)

const DEFAULT_CONFIG_PATH = `diecast.yml`
const DEFAULT_SERVE_ADDRESS = `127.0.0.1`
const DEFAULT_SERVE_PORT = 28419
const DEFAULT_ROUTE_PREFIX = `/`

type Server struct {
	Address       string
	Port          int
	MountProxy    *MountProxy
	Config        Config
	ConfigPath    string
	DefaultEngine string
	RootPath      string
	RoutePrefix   string
	router        *httprouter.Router
	server        *negroni.Negroni
}

func NewServer() *Server {
	return &Server{
		Address:     DEFAULT_SERVE_ADDRESS,
		Port:        DEFAULT_SERVE_PORT,
		ConfigPath:  DEFAULT_CONFIG_PATH,
		MountProxy:  &MountProxy{},
		RoutePrefix: DEFAULT_ROUTE_PREFIX,
	}
}

func (self *Server) Initialize() error {
	if data, err := ioutil.ReadFile(path.Join(self.RootPath, self.ConfigPath)); err == nil {
		if config, err := LoadConfig(data); err == nil {
			self.Config = config

			if err := self.InitializeMounts(config.Mounts); err != nil {
				return fmt.Errorf("Failed to initialize mounts: %v", err)
			}
		} else {
			return fmt.Errorf("Cannot load configuration at %s: %v", self.ConfigPath, err)
		}
	}

	// always make sure the root path is absolute
	if v, err := filepath.Abs(self.RootPath); err == nil {
		cwd, err := os.Getwd()

		if v == `./` && err == nil {
			self.RootPath = cwd
		} else {
			self.RootPath = v
		}
	} else {
		return err
	}

	self.RoutePrefix = strings.TrimSuffix(self.RoutePrefix, `/`)
	self.MountProxy.Server = self
	self.MountProxy.Fallback = http.Dir(self.RootPath)
	self.MountProxy.TemplatePatterns = self.Config.TemplatePatterns

	if self.MountProxy.TemplatePatterns != nil {
		log.Debugf("MountProxy: templates only apply to: %s", strings.Join(self.MountProxy.TemplatePatterns, `, `))
	}

	if err := self.setupServer(); err != nil {
		return err
	}

	return nil
}

func (self *Server) Serve() error {
	self.server.Run(fmt.Sprintf("%s:%d", self.Address, self.Port))
	return nil
}

func (self *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var templateName string

	if strings.HasSuffix(req.URL.Path, `/`) {
		templateName = fmt.Sprintf("%s%s", req.URL.Path, `index.html`)
	} else {
		templateName = path.Base(req.URL.Path)
	}

	if file, err := self.MountProxy.Open(templateName); err == nil {
		if found, err := self.RenderTemplateFromRequest(templateName, file, w, req); found {
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		} else {
			http.FileServer(self.MountProxy).ServeHTTP(w, req)
		}
	} else {
		http.Error(w, err.Error(), http.StatusNotFound)
	}
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

func (self *Server) setupServer() error {
	self.server = negroni.New()

	// setup panic recovery handler
	self.server.Use(negroni.NewRecovery())

	// setup internal/metadata routes
	mux := http.NewServeMux()

	mux.HandleFunc(fmt.Sprintf("%s/_diecast", self.RoutePrefix), func(w http.ResponseWriter, req *http.Request) {
		if data, err := json.Marshal(self); err == nil {
			w.Header().Set(`Content-Type`, `application/json`)

			if _, err := w.Write(data); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	mux.HandleFunc(fmt.Sprintf("%s/_bindings", self.RoutePrefix), func(w http.ResponseWriter, req *http.Request) {
		if data, err := json.Marshal(self.Config.Bindings); err == nil {
			w.Header().Set(`Content-Type`, `application/json`)

			if _, err := w.Write(data); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// all other routes proxy to this http.Handler
	mux.HandleFunc(`/`, func(w http.ResponseWriter, req *http.Request) {
		self.ServeHTTP(w, req)
	})

	self.server.UseHandler(mux)

	return nil
}