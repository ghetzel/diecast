package diecast

import "net/http"

var FrontMatterSeparator = []byte("---\n")
var MaxFrontMatterSize = 32768

type TemplateRenderer struct{}

func (self *TemplateRenderer) Render(w http.ResponseWriter, cfg *RendererConfig) error {
	defer cfg.Data.Close()

	if _, frontMatter, err := SplitFrontMatter(cfg.Data); err == nil {
		frontMatter.Request = cfg.Request
		return ErrNotImplemented
	} else {
		return err
	}
}
