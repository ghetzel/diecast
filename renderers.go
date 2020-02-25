package diecast

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

var renderers = make(map[string]Renderer)

func init() {
	RegisterRenderer(`pdf`, new(PdfRenderer))
	RegisterRenderer(`markdown`, new(MarkdownRenderer))
	RegisterRenderer(`sass`, new(SassRenderer))
	RegisterRenderer(`html`, new(TemplateRenderer))
	RegisterRenderer(``, new(TemplateRenderer))
}

type RenderOptions struct {
	Header        *TemplateHeader
	HeaderOffset  int
	FunctionSet   FuncMap
	Input         io.ReadCloser
	Fragments     FragmentSet
	Data          map[string]interface{}
	MimeType      string
	RequestedPath string
	Timeout       time.Duration
}

type PrewriteFunc func(*http.Request)

type Renderer interface {
	SetPrewriteFunc(PrewriteFunc)
	SetServer(*Server)
	Render(w http.ResponseWriter, req *http.Request, options RenderOptions) error
	ShouldPrerender() bool
}

func RegisterRenderer(name string, renderer Renderer) {
	renderers[name] = renderer
}

func GetRenderer(name string, server *Server) (Renderer, error) {
	if renderer, ok := renderers[name]; ok && renderer != nil {
		renderer.SetServer(server)

		return renderer, nil
	} else {
		return nil, fmt.Errorf("Unknown renderer %q", name)
	}
}

func GetRendererForFilename(filename string, server *Server) (Renderer, bool) {
	if server != nil && len(server.RendererMappings) > 0 {
		var ext = filepath.Ext(filename)
		ext = strings.TrimPrefix(ext, `.`)

		if rname, ok := server.RendererMappings[ext]; ok {
			if renderer, err := GetRenderer(rname, server); err == nil {
				return renderer, true
			}
		}
	}

	return nil, false
}
