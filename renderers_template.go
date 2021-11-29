package diecast

import "io/fs"

var FrontMatterSeparator = []byte("---\n")
var MaxFrontMatterSize = 32768
var DefaultLayoutName = `default`

type TemplateRenderer struct{}

func (self *TemplateRenderer) Render(ctx *Context, input fs.File, cfg *RendererConfig) error {
	defer input.Close()

	if tmpl, err := ParseTemplate(input); err == nil {
		if err := tmpl.LoadRelatedTemplates(ctx); err != nil {
			return err
		}

		if _, err := tmpl.DataSources.Retrieve(ctx); err != nil {
			return err
		}

		return tmpl.Render(ctx, ctx)
	} else {
		return err
	}
}
