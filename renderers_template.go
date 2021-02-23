package diecast

import (
	"io"
	"net/http"
)

var FrontMatterSeparator = []byte("---\n")
var MaxFrontMatterSize = 32768

type TemplateRenderer struct{}

func (self *TemplateRenderer) Render(ctx *Context, input http.File, cfg *RendererConfig) error {
	defer input.Close()

	if tmpl, unread, err := ParseTemplate(input); err == nil {
		// nothing was left unread
		if unread == nil {
			return tmpl.Render(ctx, nil)
		} else {
			var _, err = io.Copy(ctx, input)
			return err
		}
	} else {
		return err
	}
}
