package diecast

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/ghetzel/go-stockutil/typeutil"
)

// Perform the retrieval phase of handling a request.
func (self *Server) serveHttpPhaseRetrieve(ctx *Context, req *http.Request) (http.File, error) {
	if err := self.prep(); err != nil {
		return nil, err
	}

	var file http.File
	var lerr error

	for _, tryPath := range self.retrieveTryPaths(req) {
		// ctx.Debugf("retrieve: try path %v", tryPath)
		file, lerr = self.VFS.Open(tryPath)

		if lerr == nil {
			// ctx.Debugf("retrieve: path %v succeeded", tryPath)
			break
		}
	}

	// last results from Open() are what we return, error or not
	return file, lerr
}

// builds a list of filesystem objects to search for in response to the request URL path
func (self *Server) retrieveTryPaths(req *http.Request) (paths []string) {
	paths = append(paths, req.URL.Path)

	if strings.HasSuffix(req.URL.Path, `/`) {
		paths = append(paths, filepath.Join(
			req.URL.Path,
			typeutil.OrString(self.Paths.IndexFilename, DefaultIndexFilename),
		))
	}

	return
}
