package diecast

import (
	"github.com/ghetzel/diecast/v2/internal"
	"io/fs"
)

type TemplateRenderer struct {
}

func (self *TemplateRenderer) Render(ctx *Context, input fs.File, cfg *RendererConfig) error {
	defer input.Close()

	var _, funcs = internal.GetFunctions(ctx.Server())

	if tmpl, err := ParseTemplateWithFuncs(input, funcs); err == nil {
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
