package diecast

import (
	"net/http"
)

var FrontMatterSeparator = []byte("---\n")
var MaxFrontMatterSize = 32768

type TemplateRenderer struct{}

func (self *TemplateRenderer) Render(ctx *Context, input http.File, cfg *RendererConfig) error {
	defer input.Close()

	if tmpl, err := ParseTemplate(input); err == nil {
		return tmpl.Render(ctx, nil)
	} else {
		return err
	}
}
