package diecast

import (
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/typeutil"
)

var DefaultIndexFilename = `index.html`

type Server struct {
	ServePath     http.Dir
	IndexFilename string
	vfs           VFS
	ovfs          http.FileSystem
}

func (self *Server) Open(name string) (http.File, error) {
	return self.vfs.Open(name)
}

func (self *Server) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var file http.File
	var err error

	// VALIDATE
	// -------------------------------------------------------------------------------------------------------------------
	//  ▶ decide to accept/reject request
	//  ▶ perform authentication checks
	//  ▶ any other security or data validation before causing a VFS retrieval
	//
	err = self.Validate(req)

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
	} else if p := self.ServePath; p != `` {
		self.vfs.SetFallbackFS(p)
	} else {
		self.vfs.SetFallbackFS(http.Dir(`.`))
	}

	return nil
}

// Perform the validation phase of handling a request.
func (self *Server) Validate(req *http.Request) error {
	if err := self.prep(); err != nil {
		return err
	}

	return nil
}

// Perform the retrieval phase of handling a request.
func (self *Server) Retrieve(req *http.Request) (http.File, error) {
	if err := self.prep(); err != nil {
		return nil, err
	}

	var file http.File
	var lerr error

	for _, tryPath := range self.retrieveTryPaths(req) {
		log.Debugf("trying: %s", tryPath)
		file, lerr = self.vfs.Open(tryPath)

		if lerr == nil {
			break
		}
	}

	return file, lerr
}

// builds a list of filesystem objects to search for in response to the request URL path
func (self *Server) retrieveTryPaths(req *http.Request) (paths []string) {
	paths = append(paths, req.URL.Path)

	if strings.HasSuffix(req.URL.Path, `/`) {
		paths = append(paths, filepath.Join(
			req.URL.Path,
			typeutil.OrString(self.IndexFilename, DefaultIndexFilename),
		))
	}

	return
}

// Render a retrieved file to the given response writer.
func (self *Server) Render(w http.ResponseWriter, req *http.Request, source http.File) error {
	_, err := io.Copy(w, source)
	return err
}
