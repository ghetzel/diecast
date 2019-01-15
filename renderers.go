package diecast

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

type RenderOptions struct {
	Header        *TemplateHeader
	HeaderOffset  int
	FunctionSet   FuncMap
	Input         io.ReadCloser
	Data          map[string]interface{}
	MimeType      string
	RequestedPath string
	HasLayout     bool
	Timeout       time.Duration
}

type Renderer interface {
	Render(w http.ResponseWriter, req *http.Request, options RenderOptions) error
	ShouldPrerender() bool
}

func GetRenderer(name string, server *Server) (Renderer, error) {
	switch name {
	case `pdf`:
		return &PdfRenderer{server: server}, nil
	case `markdown`:
		return &MarkdownRenderer{server: server}, nil
	case `sass`:
		return &SassRenderer{server: server}, nil
	case ``, `html`:
		return &TemplateRenderer{server: server}, nil
	default:
		return nil, fmt.Errorf("Unknown renderer %q", name)
	}
}

func GetRendererForFilename(filename string, server *Server) (Renderer, bool) {
	if server != nil && len(server.RendererMappings) > 0 {
		ext := filepath.Ext(filename)
		ext = strings.TrimPrefix(ext, `.`)

		if rname, ok := server.RendererMappings[ext]; ok {
			if renderer, err := GetRenderer(rname, server); err == nil {
				return renderer, true
			}
		}
	}

	return nil, false
}
