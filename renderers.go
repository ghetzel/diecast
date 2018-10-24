package diecast

import (
	"fmt"
	"io"
	"net/http"
)

type RenderOptions struct {
	Header        *TemplateHeader
	HeaderOffset  int
	FunctionSet   FuncMap
	Input         io.ReadCloser
	MimeType      string
	RequestedPath string
	HasLayout     bool
}

type Renderer interface {
	Render(w http.ResponseWriter, req *http.Request, options RenderOptions) error
}

func GetRenderer(name string, server *Server) (Renderer, error) {
	switch name {
	case `pdf`:
		return &PdfRenderer{server: server}, nil
	case `markdown`:
		return &MarkdownRenderer{server: server}, nil
	case ``:
		return &TemplateRenderer{server: server}, nil
	default:
		return nil, fmt.Errorf("Unknown renderer %q", name)
	}
}
