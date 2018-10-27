package diecast

import (
	"io/ioutil"
	"net/http"

	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday"
)

type MarkdownRenderer struct {
	server *Server
}

func (self *MarkdownRenderer) ShouldPrerender() bool {
	return true
}

func (self *MarkdownRenderer) Render(w http.ResponseWriter, req *http.Request, options RenderOptions) error {
	defer options.Input.Close()

	if input, err := ioutil.ReadAll(options.Input); err == nil {
		output := blackfriday.MarkdownCommon(input)
		output = bluemonday.UGCPolicy().SanitizeBytes(output)

		w.Header().Set(`Content-Type`, `text/html; charset=utf-8`)
		_, err := w.Write(output)
		return err
	} else {
		return err
	}
}
