package diecast

import (
	"io/fs"
)

type TemplateRenderer struct {
}

func (self *TemplateRenderer) Render(ctx *Context, input fs.File, cfg *RendererConfig) error {
	defer input.Close()

	if tmpl, err := ParseTemplate(input); err == nil {
		if err := tmpl.LoadRelatedTemplates(ctx); err != nil {
			return err
		}

		// TODO: tmpl.Funcs() here

		if _, err := tmpl.DataSources.Retrieve(ctx); err != nil {
			return err
		}

		return tmpl.Render(ctx, ctx)
	} else {
		return err
	}
}
