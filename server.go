package diecast

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"time"

	"github.com/ghetzel/go-stockutil/httputil"
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
	DataSources   DataSet           `yaml:"dataSources"`
	Paths         ServerPaths       `yaml:"paths"`
	Validators    []ValidatorConfig `yaml:"validators"`
	Renderers     []RendererConfig  `yaml:"renderers"`
	VerifyMethod  string            `yaml:"verifyMethod"`
	VerifyPath    string            `yaml:"verifyPath"`
	VerifyTimeout string            `yaml:"verifyTimeout"`
	VFS           VFS               `yaml:"vfs"`
	ovfs          fs.FS
	startFuncs    []ServerStartFunc
}

// Loads a YAML-formatted configuration from the given reader and returns a Server.
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

// Loads a YAML-formatted configuration from the given path.  The Server that is returned will
// have its root directory set relative to the parent directory of the configuration file (unless
// otherwise configured).
func NewServerFromFile(cfgfile string) (*Server, error) {
	if cfg, err := os.Open(cfgfile); err == nil {
		defer cfg.Close()

		if srv, err := NewServerFromConfig(cfg); err == nil {
			srv.VFS.SetFallbackFS(os.DirFS(filepath.Dir(cfgfile)))
			return srv, nil
		} else {
			return nil, err
		}
	} else {
		return nil, err
	}
}

// Configure VFS layers from a connection string
func (self *Server) LoadLayersFromString(specs ...string) error {
	for _, spec := range specs {
		if layer, err := LayerFromString(spec); err == nil {
			self.VFS.Layers = append(self.VFS.Layers, *layer)
		} else {
			return err
		}
	}

	return nil
}

// Perform a pre-configured request that must succeed to be considered successful.
func (self *Server) Verify() error {
	var res, err = self.SimulateRequest(
		typeutil.OrString(self.VerifyMethod, DefaultVerifyMethod),
		typeutil.OrString(self.VerifyPath, DefaultVerifyPath),
		nil,
		nil,
		nil,
	)

	if rc := res.Body; rc != nil {
		rc.Close()
	}

	for _, deepError := range res.Header.Values(XDiecastError) {
		err = log.AppendError(err, errors.New(deepError))
	}

	return err
}

// Add a handler function that will be called when the server starts.
func (self *Server) OnStart(fn ServerStartFunc) {
	if fn != nil {
		self.startFuncs = append(self.startFuncs, fn)
	}
}

// Simulates a single request, returning the http.Response that would be sent to a client, and an error should one occur.
func (self *Server) SimulateRequest(method string, path string, body io.Reader, qs map[string]interface{}, header map[string]interface{}) (*http.Response, error) {
	var wr = httptest.NewRecorder()
	var req = httptest.NewRequest(
		typeutil.OrString(method, http.MethodGet),
		path,
		body,
	)

	for k, v := range qs {
		httputil.SetQ(req.URL, k, typeutil.Auto(v))
	}

	for k, v := range header {
		req.Header.Set(k, typeutil.String(v))
	}

	self.ServeHTTP(wr, req)
	wr.Flush()

	if code := wr.Code; code < 400 {
		return wr.Result(), nil
	} else {
		return wr.Result(), fmt.Errorf("HTTP %d: %s", code, http.StatusText(code))
	}
}

// Start accepting connections and responding to requests on the given address.
func (self *Server) ListenAndServe(address string) error {
	var errchan = make(chan error)
	var readychan = make(chan bool)
	var hsrv = &http.Server{
		Addr:    typeutil.OrString(address, self.Address, DefaultAddress),
		Handler: self,
	}

	// verification check happens in a goroutine BEFORE the server starts listening
	go func() {
		var ok bool

		// tell the listen goroutine whether we're ready for it to start listening or to exit
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

			// verify happens in yet another goroutine so we can implement a timeout (below)
			go func() {
				verr <- self.Verify()
			}()

			select {
			case err := <-verr:
				if err == nil {
					log.Debugf("verify: ok")
					ok = true // ok to start listening
				} else {
					// log.Debugf("verify: error %v", err)
					errchan <- err
				}
			case <-time.After(verifyTimeout):
				errchan <- fmt.Errorf("timed out waiting for verify test")
			}
		} else {
			ok = true // ok to start listening
		}
	}()

	// wait for an ok signal then start listening blocked in another goroutine
	go func() {
		if r := <-readychan; r {
			log.Noticef("listening on %v", hsrv.Addr)
			errchan <- hsrv.ListenAndServe()
		}
	}()

	// whoever errors first returns
	return <-errchan
}

// Implements the http.Handler interface.
func (self *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var file fs.File
	var err error
	var ctx = NewContext(self)

	// populate the context with data from the global DataSet
	if _, err := self.DataSources.Retrieve(ctx); err != nil {
		ctx.Warningf("data: %v", err)
		self.writeResponse(ctx, err)
		return
	}

	if err := self.prep(); err != nil {
		ctx.Warningf("prep: %v", err)
		self.writeResponse(ctx, err)
		return
	}

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

	// END.OF.LINE.
	// -------------------------------------------------------------------------------------------------------------------
}

// Intelligently respond in a consistent manner with the data provided, including error detection, redirection,
// and status code enforcement.
//
// When data is nil, and no code is provided -> HTTP 204
// When data is an error, write out the error text and ensure the HTTP status is >= 400
// When code is [300,399], an HTTP redirect will occur, redirecting to the path resulting from stringifying data.
// When data is a Map or Array, it will be encoded and returned as JSON with Content-Type: application/json.
//
// All other conditions will convert the data to []byte and write that out directly.
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

		ctx.Header().Add(XDiecastError, err.Error())
	}

	// auto-jsonify complex types
	if typeutil.IsMap(data) || typeutil.IsArray(data) {
		if b, mimetype, err := AutoencodeByFilename(req.URL.Path, data); err == nil {
			data = b
			ctx.SetTypeHint(mimetype)
		} else {
			data = err.Error()
			httpStatus = http.StatusInternalServerError
			ctx.Header().Add(XDiecastError, err.Error())
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
