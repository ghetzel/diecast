package diecast

import (
	"net/http"

	"github.com/ghetzel/go-stockutil/typeutil"
)

var DefaultIndexFilename = `index.html`
var DefaultLayoutsDir = `/_layouts`
var DefaultErrorsDir = `/_errors`

type ServerPaths struct {
	Root          http.Dir `yaml:"root"`
	LayoutsDir    string   `yaml:"layouts"`
	ErrorsDir     string   `yaml:"errors"`
	IndexFilename string   `yaml:"indexFilename"`
}

type Server struct {
	Paths      ServerPaths       `yaml:"paths"`
	Validators []ValidatorConfig `yaml:"validators"`
	vfs        VFS
	ovfs       http.FileSystem
}

// Implements the http.FileSystem interface.
func (self *Server) Open(name string) (http.File, error) {
	return self.vfs.Open(name)
}

// Implements the http.Handler interface.
func (self *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var file http.File
	var err error

	// VALIDATE
	// -------------------------------------------------------------------------------------------------------------------
	//  ▶ decide to accept/reject request
	//  ▶ perform authentication checks
	//  ▶ any other security or data validation before causing a VFS retrieval
	//
	err = self.ValidateAll(req)

	if err != nil {
		self.writeResponse(w, req, err)
		return
	}

	// RETRIEVE
	// -------------------------------------------------------------------------------------------------------------------
	//  ▶ use the request to locate the data and metadata that will be rendered into a response
	//
	file, err = self.Retrieve(req)

	if err == nil {
		defer file.Close()
	} else {
		self.writeResponse(w, req, err)
		return
	}

	// RENDER
	// -------------------------------------------------------------------------------------------------------------------
	//  ▶ consume the input data found in RETRIEVE and write whatever response the requestor will receive
	//
	err = self.Render(w, req, file)

	if err == nil {
		return
	} else {
		self.writeResponse(w, req, err)
		return
	}
}

func (self *Server) writeResponse(w http.ResponseWriter, req *http.Request, data interface{}, code ...int) {
	var httpStatus int = http.StatusOK

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
		http.Redirect(w, req, typeutil.OrString(data, `/`), httpStatus)
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

	w.WriteHeader(httpStatus)

	if data != nil {
		w.Write(typeutil.Bytes(data))
	}
}

// setup and populate any last-second things we might need to process a request
func (self *Server) prep() error {
	if self.ovfs != nil {
		self.vfs.SetFallbackFS(self.ovfs)
	} else if p := self.Paths.Root; p != `` {
		self.vfs.SetFallbackFS(p)
	} else {
		self.vfs.SetFallbackFS(http.Dir(`.`))
	}

	return nil
}
