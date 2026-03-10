package diecast

import "net/http"

func init() {
	// RegisterRendererType(`pdf`, new(TemplateRenderer))
	// RegisterRendererType(`image`, new(TemplateRenderer))
	// RegisterRendererType(`msoffice`, new(TemplateRenderer))
	RegisterRendererType(`template`, new(TemplateRenderer))
	RegisterRendererType(``, new(PassthroughRenderer))

	// setup default type handlers
	RegisterRendererByMIME(`text/html`, RendererConfig{
		Type: `template`,
		Methods: []string{
			http.MethodGet,
		},
	})

	RegisterRendererByMIME(`application/xml`, RendererConfig{
		Type: `template`,
		Methods: []string{
			http.MethodGet,
		},
	})
}
