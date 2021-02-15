package diecast

import (
	"net/http"

	"github.com/ghetzel/go-stockutil/typeutil"
)

var DefaultIndexFilename = `index.html`
var DefaultLayoutsDir = `/_layouts`
var DefaultErrorsDir = `/_errors`

type ServerPaths struct {
	LayoutsDir    string `yaml:"layouts"`
	ErrorsDir     string `yaml:"errors"`
	IndexFilename string `yaml:"indexFilename"`
}

type Server struct {
	Paths      ServerPaths       `yaml:"paths"`
	Validators []ValidatorConfig `yaml:"validators"`
	VFS        VFS
	ovfs       http.FileSystem
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

// Intelligently respond in a consistent manner with the data provided, including error detection, redirection,
// and status code enforcement.
//
//   If data is nil, and no code is provided -> HTTP 204
//   If data is an error, write out the error text and ensure the HTTP status is >= 400
//   If code is [300,399], an HTTP redirect will occur, redirecting to the path resulting from stringifying data.
//   If data is a Map or Array, I will be encoded and returned as JSON with Content-Type: application/json.
//	 All other conditions will convert the data to []byte and write that out directly.
func (self *Server) writeResponse(w http.ResponseWriter, req *http.Request, data interface{}, code ...int) {
	var httpStatus int = http.StatusOK

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

	// auto-jsonify complex types
	if typeutil.IsMap(data) || typeutil.IsArray(data) {
		if b, mimetype, err := AutoencodeByFilename(req.URL.Path, data); err == nil {
			data = b
			w.Header().Set(`Content-Type`, mimetype)
		} else {
			data = err.Error()
			httpStatus = http.StatusInternalServerError
		}
	}

	// commit to responding and write out data
	w.WriteHeader(httpStatus)

	if data != nil {
		w.Write(typeutil.Bytes(data))
	}
}

// setup and populate any last-second things we might need to process a request
func (self *Server) prep() error {
	if self.ovfs != nil {
		self.VFS.SetFallbackFS(self.ovfs)
	}

	return nil
}
