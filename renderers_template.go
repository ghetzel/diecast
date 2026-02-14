package diecast

import (
	"io/fs"

	"github.com/ghetzel/diecast/v2/internal"
)

type TemplateRenderer struct {
}

func (self *TemplateRenderer) Render(ctx *Context, input fs.File, cfg *RendererConfig) error {
	defer input.Close()

	var _, funcs = internal.GetFunctionsWithRequest(ctx.Server(), ctx)

	if tmpl, err := ParseTemplateWithFuncs(input, funcs); err == nil {
		ctx.isLegacyV1 = tmpl.IsLegacyV1

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
