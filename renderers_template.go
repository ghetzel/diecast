package diecast

import (
	"io"
	"net/http"
)

var FrontMatterSeparator = []byte("---\n")
var MaxFrontMatterSize = 32768

type TemplateRenderer struct{}

func (self *TemplateRenderer) Render(w http.ResponseWriter, cfg *RendererConfig) error {
	var data = cfg.Data()
	defer data.Close()

	if tmpl, unread, err := ParseTemplate(data); err == nil {
		// nothing was left unread
		if unread == nil {
			return tmpl.Render(w)
		} else {
			var _, err = io.Copy(w, data)
			return err
		}
	} else {
		return err
	}
}
