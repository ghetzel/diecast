package diecast

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/ghetzel/go-stockutil/typeutil"
)

var FrontMatterSeparator = []byte("---\n")
var MaxFrontMatterSize = 32768
var DefaultLayoutName = `default`

type TemplateRenderer struct{}

func (self *TemplateRenderer) Render(ctx *Context, input http.File, cfg *RendererConfig) error {
	defer input.Close()

	if tmpl, err := ParseTemplate(input); err == nil {
		var doLayout bool = true
		var layoutName = ctx.T(tmpl.Layout).OrString(DefaultLayoutName)
		var lext string = typeutil.OrString(filepath.Ext(layoutName), `.html`)

		switch strings.ToLower(layoutName) {
		case `none`, `false`:
			doLayout = false
		}

		if doLayout {
			var layoutPath = filepath.Join(
				typeutil.OrString(ctx.Server().Paths.LayoutsDir, DefaultLayoutsDir),
				layoutName+lext,
			)

			// tell the template
			if layoutFile, err := ctx.Open(layoutPath); err == nil {
				defer layoutFile.Close()

				if err := tmpl.AttachFile(`layout:`+layoutName, layoutFile); err == nil {
					tmpl.EntryPoint = `layout:` + layoutName
				} else {
					return err
				}
			} else if layoutName != DefaultLayoutName {
				return err
			}
		}

		if _, err := tmpl.DataSources.Retrieve(ctx); err != nil {
			return err
		}

		return tmpl.Render(ctx, ctx)
	} else {
		return err
	}
}
