package diecast

import (
	"io"
	"net/http"

	"github.com/microcosm-cc/bluemonday"
	blackfriday "github.com/russross/blackfriday/v2"
)

type MarkdownRenderer struct {
	server   *Server
	prewrite PrewriteFunc
}

func (renderer *MarkdownRenderer) ShouldPrerender() bool {
	return true
}

func (renderer *MarkdownRenderer) SetServer(server *Server) {
	renderer.server = server
}

func (renderer *MarkdownRenderer) SetPrewriteFunc(fn PrewriteFunc) {
	renderer.prewrite = fn
}

func (renderer *MarkdownRenderer) Render(w http.ResponseWriter, req *http.Request, options RenderOptions) error {
	defer options.Input.Close()

	if input, err := io.ReadAll(options.Input); err == nil {
		var output = blackfriday.Run(
			input,
			blackfriday.WithExtensions(blackfriday.CommonExtensions),
		)

		output = bluemonday.UGCPolicy().SanitizeBytes(output)

		w.Header().Set(`Content-Type`, `text/html; charset=utf-8`)

		if fn := renderer.prewrite; fn != nil {
			fn(req)
		}

		_, err := w.Write(output)
		return err
	} else {
		return err
	}
}
