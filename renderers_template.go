package diecast

import (
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/ghetzel/diecast/v2/internal"
)

type TemplateRenderer struct {
}

func (self *TemplateRenderer) Render(ctx *Context, input fs.File, cfg *RendererConfig) error {
	defer input.Close()

	var _, funcs = internal.GetFunctionsWithRequest(ctx.Server(), ctx)

	if tmpl, err := ParseTemplateWithFuncs(input, funcs); err == nil {
		ctx.isLegacyV1 = tmpl.IsLegacyV1

		for k, v := range tmpl.Page {
			ctx.SetValue(`page.`+k, v)
		}

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

func (self *TemplateRenderer) ShouldApplyTo(req *http.Request) bool {
	var filename = filepath.Base(req.URL.Path)

	if strings.HasPrefix(filename, `_`) {
		return false
	}

	return true
}
