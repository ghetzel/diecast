package diecast

import (
	"io/ioutil"
	"net/http"

	"github.com/microcosm-cc/bluemonday"
	blackfriday "github.com/russross/blackfriday/v2"
)

type MarkdownRenderer struct {
	server   *Server
	prewrite PrewriteFunc
}

func (self *MarkdownRenderer) ShouldPrerender() bool {
	return true
}

func (self *MarkdownRenderer) SetServer(server *Server) {
	self.server = server
}

func (self *MarkdownRenderer) SetPrewriteFunc(fn PrewriteFunc) {
	self.prewrite = fn
}

func (self *MarkdownRenderer) Render(w http.ResponseWriter, req *http.Request, options RenderOptions) error {
	defer options.Input.Close()

	if input, err := ioutil.ReadAll(options.Input); err == nil {
		output := blackfriday.Run(
			input,
			blackfriday.WithExtensions(blackfriday.CommonExtensions),
		)

		output = bluemonday.UGCPolicy().SanitizeBytes(output)

		w.Header().Set(`Content-Type`, `text/html; charset=utf-8`)

		if fn := self.prewrite; fn != nil {
			fn(req)
		}

		_, err := w.Write(output)
		return err
	} else {
		return err
	}
}
