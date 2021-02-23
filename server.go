package diecast

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"time"

	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/typeutil"
	"gopkg.in/yaml.v2"
)

var DefaultConfigFilename = `diecast.yaml`
var DefaultAddress = `127.0.0.1:28419`
var DefaultIndexFilename = `index.html`
var DefaultLayoutsDir = `/_layouts`
var DefaultErrorsDir = `/_errors`
var DefaultVerifyMethod = `GET`
var DefaultVerifyPath = `/`
var DefaultVerifyTimeout = `1s`

type ServerStartFunc func(*Server, error) error

type ServerPaths struct {
	LayoutsDir    string `yaml:"layouts"`
	ErrorsDir     string `yaml:"errors"`
	IndexFilename string `yaml:"indexFilename"`
}

type Server struct {
	Address       string            `yaml:"address"`
	Paths         ServerPaths       `yaml:"paths"`
	Validators    []ValidatorConfig `yaml:"validators"`
	Renderers     []RendererConfig  `yaml:"renderers"`
	VerifyMethod  string            `yaml:"verifyMethod"`
	VerifyPath    string            `yaml:"verifyPath"`
	VerifyTimeout string            `yaml:"verifyTimeout"`
	VFS           VFS               `yaml:"vfs"`
	ovfs          http.FileSystem
	startFuncs    []ServerStartFunc
}

func NewServerFromConfig(r io.Reader) (*Server, error) {
	var srv Server

	if data, err := ioutil.ReadAll(r); err == nil {
		if err := yaml.UnmarshalStrict(data, &srv); err == nil {
			return &srv, nil
		} else {
			return nil, err
		}
	} else {
		return nil, err
	}
}

func NewServerFromFile(cfgfile string) (*Server, error) {
	if cfg, err := os.Open(cfgfile); err == nil {
		defer cfg.Close()

		if srv, err := NewServerFromConfig(cfg); err == nil {
			srv.VFS.SetFallbackFS(http.Dir(filepath.Dir(cfgfile)))
			return srv, nil
		} else {
			return nil, err
		}
	} else {
		return nil, err
	}
}

func (self *Server) Verify() error {
	var res, err = self.doSimpleRequest(
		typeutil.OrString(self.VerifyMethod, DefaultVerifyMethod),
		typeutil.OrString(self.VerifyPath, DefaultVerifyPath),
	)

	if rc := res.Body; rc != nil {
		rc.Close()
	}

	return err
}

func (self *Server) OnStart(fn ServerStartFunc) {
	if fn != nil {
		self.startFuncs = append(self.startFuncs, fn)
	}
}

func (self *Server) doSimpleRequest(method string, path string) (*http.Response, error) {
	var wr = httptest.NewRecorder()
	var req = httptest.NewRequest(method, path, nil)

	self.ServeHTTP(wr, req)

	wr.Flush()

	if code := wr.Code; code < 400 {
		return wr.Result(), nil
	} else {
		return wr.Result(), fmt.Errorf("HTTP %d: %s", code, http.StatusText(code))
	}
}

func (self *Server) ListenAndServe(address string) error {
	var hsrv = &http.Server{
		Addr:    typeutil.OrString(address, self.Address, DefaultAddress),
		Handler: self,
	}

	var errchan = make(chan error)
	var readychan = make(chan bool)

	go func() {
		var ok bool

		defer func(r *bool) {
			readychan <- *r
		}(&ok)

		if verifyTimeout := typeutil.Duration(
			typeutil.OrString(
				self.VerifyTimeout,
				DefaultVerifyTimeout,
			),
		); verifyTimeout > 0 {
			var verr = make(chan error)

			go func() {
				verr <- self.Verify()
			}()

			select {
			case err := <-verr:
				if err == nil {
					log.Debugf("verify: ok")
					ok = true
				} else {
					log.Debugf("verify: error %v", err)
					errchan <- err
				}
			case <-time.After(verifyTimeout):
				errchan <- fmt.Errorf("timed out waiting for verify test")
			}
		}
	}()

	go func() {
		if r := <-readychan; r {
			log.Noticef("listening on %v", hsrv.Addr)
			errchan <- hsrv.ListenAndServe()
		}
	}()

	return <-errchan
}

// Implements the http.Handler interface.
func (self *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var file http.File
	var err error
	var ctx = NewContext(&self.VFS)

	ctx.Start(w, req)
	defer ctx.Done()

	// VALIDATE
	// -------------------------------------------------------------------------------------------------------------------
	//  ▶ decide to accept/reject request
	//  ▶ perform authentication checks
	//  ▶ any other security or data validation before causing a VFS retrieval
	//
	err = self.serveHttpPhaseValidate(ctx)

	if err != nil {
		ctx.Warningf("validate: %v", err)
		self.writeResponse(ctx, err)
		return
	}

	// RETRIEVE
	// -------------------------------------------------------------------------------------------------------------------
	//  ▶ use the request to locate the data and metadata that will be rendered into a response
	//
	file, err = self.serveHttpPhaseRetrieve(ctx)

	if err == nil {
		defer file.Close()
	} else {
		ctx.Debugf("retrieve: %v", err)
		self.writeResponse(ctx, err)
		return
	}

	// flush any last-minute headers and other prep before rendering occurs
	ctx.finalizeBeforeRender()

	// RENDER
	// -------------------------------------------------------------------------------------------------------------------
	//  ▶ consume the input data found in RETRIEVE and write whatever response the requestor will receive
	//
	err = self.serveHttpPhaseRender(ctx, file)

	if err == nil {
		return
	} else {
		ctx.Debugf("render: %v", err)
		self.writeResponse(ctx, err)
		return
	}
}

// Intelligently respond in a consistent manner with the data provided, including error detection, redirection,
// and status code enforcement.
//
// When data is nil, and no code is provided -> HTTP 204
// When data is an error, write out the error text and ensure the HTTP status is >= 400
// When code is [300,399], an HTTP redirect will occur, redirecting to the path resulting from stringifying data.
// When data is a Map or Array, I will be encoded and returned as JSON with Content-Type: application/json.
//
// All other conditions will convert the data to []byte and write that out directly.
//
func (self *Server) writeResponse(ctx *Context, data interface{}, code ...int) {
	var req = ctx.Request()
	var httpStatus int = ctx.Code()

	if data == nil {
		httpStatus = http.StatusNoContent
	}

	// see if the response body itself has an opinion on what its HTTP status code should be
	if c, ok := data.(Codeable); ok {
		httpStatus = c.Code()
	}

	// honor any valid code given as an explicit override in the variadic code argument
	if len(code) > 0 && code[0] >= 100 {
		httpStatus = code[0]
	}

	// treat 3xx codes as redirects, interpreting data as the new location string
	if httpStatus >= 300 && httpStatus < 400 {
		http.Redirect(ctx, req, typeutil.OrString(data, `/`), httpStatus)
		return
	}

	// extract error data
	if err, ok := data.(error); ok && err != nil {
		data = err.Error()

		// if we're returning an error, do not permit non-error response statuses
		if httpStatus < 400 {
			httpStatus = http.StatusInternalServerError
		}
	}

	// auto-jsonify complex types
	if typeutil.IsMap(data) || typeutil.IsArray(data) {
		if b, mimetype, err := AutoencodeByFilename(req.URL.Path, data); err == nil {
			data = b
			ctx.SetTypeHint(mimetype)
		} else {
			data = err.Error()
			httpStatus = http.StatusInternalServerError
		}
	}

	// commit to responding and write out data
	ctx.WriteHeader(httpStatus)

	if data != nil {
		ctx.Write(typeutil.Bytes(data))
	}
}

// setup and populate any last-second things we might need to process a request
func (self *Server) prep() error {
	if self.ovfs != nil {
		self.VFS.SetFallbackFS(self.ovfs)
	}

	return nil
}
