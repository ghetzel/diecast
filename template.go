package diecast

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/ghetzel/diecast/v2/internal"
	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/typeutil"
	"github.com/pkg/errors"
)

var DefaultEntryPoint = `content`
var DefaultLayoutName = `default`
var DefaultTemplateEngine = `text`
var Delimiters = internal.Delimiters
var FrontMatterSeparator = internal.FrontMatterSeparator
var LayoutNamePrefix string = `layout:`
var MaxFrontMatterSize = 32768
var DefaultIncludesDir = `_includes`

type FuncMap = internal.FuncMap

type Template struct {
	*internal.TemplateHeader
	body     []byte
	buf      *bytes.Buffer
	gotmpl   *internal.GolangTemplate
	initDone bool
	funcs    FuncMap
	ctx      *Context
}

// Parse a template from the given string.
func ParseTemplateString(source string) (*Template, error) {
	return ParseTemplate(bytes.NewBufferString(source))
}

// Parse a template by reading it from the given reader.
func ParseTemplate(source io.Reader) (*Template, error) {
	return ParseTemplateWithFuncs(source, nil)
}

// Parse a template by reading it from the given reader, exposing the given funcs to the template engine.
func ParseTemplateWithFuncs(source io.Reader, funcs FuncMap) (*Template, error) {
	if source == nil {
		return nil, io.EOF
	}

	var tmpl = new(Template)

	if hdr, body, err := internal.SplitTemplateHeaderContent(source); err == nil {
		tmpl.funcs = funcs
		tmpl.TemplateHeader = hdr
		tmpl.body = body
	} else {
		return nil, err
	}

	return tmpl, tmpl.init()
}

// Render the template on read. Implements the io.Reader interface.
func (template *Template) Read(b []byte) (int, error) {
	if err := template.init(); err == nil {
		if template.buf == nil {
			var dst bytes.Buffer

			if err := template.Render(template.ctx, &dst); err != nil {
				return 0, err
			}

			template.buf = &dst
		}
	} else {
		return 0, fmt.Errorf("init err: %v", err)
	}

	if buf := template.buf; buf != nil {
		return buf.Read(b)
	} else {
		return 0, io.EOF
	}
}

// Implement io.Closer
func (template *Template) Close() error {
	template.initDone = false
	template.buf = nil
	template.gotmpl = nil
	return nil
}

// Embed a Context to use as initial state for Render()
func (template *Template) SetContext(ctx *Context) {
	template.ctx = ctx
}

func (template *Template) reinit() error {
	template.initDone = false
	return template.init()
}

// Initialize the template, parsing the data and making the object ready for subsequent calls to Render
func (template *Template) init() error {
	if template.initDone {
		return nil
	}

	var engine = typeutil.OrString(template.Engine, DefaultTemplateEngine)
	// var name = typeutil.OrString(template.Filename, engine+`:`+template.SHA512SUM)

	if gotmpl, err := internal.ParseGolangTemplate(
		template.entryPoint(),
		engine,
		template.TemplateString(),
		template.funcs,
	); err == nil {
		template.gotmpl = gotmpl
		template.initDone = true
		return nil
	} else {
		fmt.Printf("gotmpl: %v\n", err)
		return err
	}
}

func (template *Template) Funcs(funcMap internal.FuncMap) *Template {
	template.funcs = funcMap
	template.reinit()

	return template
}

// Return the raw, unrendered template source.
func (template *Template) TemplateString() string {
	return string(template.body)
}

// Render the template when being converted to a string. Implements fmt.Stringer.
func (template *Template) String() string {
	if err := template.init(); err == nil {
		var dst bytes.Buffer

		if err := template.Render(template.ctx, &dst); err == nil {
			switch out := dst.String(); out {
			case `<no value>`:
				return ``
			default:
				return out
			}
		} else {
			return fmt.Sprintf("<!-- TEMPLATE ERROR: %v -->", err)
		}
	} else {
		return ``
	}
}

func (template *Template) entryPoint() string {
	return typeutil.OrString(template.EntryPoint, DefaultEntryPoint)
}

// Refresh all data sources and render the template, writing the results to the giveni io.Writer.
func (template *Template) Render(ctx *Context, w io.Writer) error {
	if err := template.init(); err != nil {
		return err
	}

	if ctx == nil {
		ctx = NewContext(nil)
	}

	if err := template.applyPageVars(ctx); err != nil {
		return err
	}

	if w == nil {
		w = ctx
	}

	ctx.Debugf("template: known templates: %s", strings.Join(template.gotmpl.Names(), `, `))
	ctx.Debugf("template: entrypoint: %s", template.entryPoint())

	return template.gotmpl.ExecuteTemplate(w, template.entryPoint(), ctx.Data())
}

// Returns the SHA512 checksum of the underlying template file.
func (template *Template) Checksum() string {
	return template.SHA512SUM
}

func (template *Template) applyPageVars(ctx *Context) error {
	if template.Page == nil {
		template.Page = make(map[string]any)
	}

	template.Page[`_`] = map[string]any{
		`id`: ctx.ID(),
	}

	var pageVars = maputil.Apply(template.Page, func(key []string, value any) (any, bool) {
		if mii, ok := value.(map[any]any); ok {
			var msi = make(map[string]any)
			for ki, vi := range mii {
				msi[typeutil.String(ki)] = vi
			}

			return msi, true
		} else if ev, err := ctx.Eval(value); err == nil {
			return ev, true
		} else {
			return value, false
		}
	})

	if curPageVars := ctx.Get(`page`); curPageVars.IsMap() {
		if m, err := maputil.Merge(curPageVars, pageVars); err == nil {
			pageVars = m
		} else {
			return err
		}
	}

	ctx.SetValue(`page`, pageVars)
	return nil
}

func (template *Template) attachTemplate(ctx *Context, tmplName string, r io.Reader) error {
	if tmpl, err := ParseTemplateWithFuncs(r, template.funcs); err == nil {
		if err := tmpl.LoadRelatedTemplates(ctx); err != nil {
			return fmt.Errorf("%s: %v", tmplName, err)
		}

		// whatever we need to do to merge in the new template header, do it here
		template.EntryPoint = typeutil.OrString(tmpl.EntryPoint, template.EntryPoint)
		template.DataSources = append(tmpl.DataSources, template.DataSources...)

		for k, v := range template.ResponseHeaders {
			ctx.Header().Set(k, typeutil.String(v))
		}

		// add this new data to our existing template tree and return
		if pt := tmpl.gotmpl.ParseTree(); pt != nil {
			var _, err = template.gotmpl.AddParseTree(tmplName, pt)
			return err
		} else {
			return fmt.Errorf("invalid layout template")
		}
	} else {
		return err
	}
}

func (template *Template) layoutName(name string) string {
	return LayoutNamePrefix + strings.TrimPrefix(name, LayoutNamePrefix)
}

func (template *Template) LoadRelatedTemplates(ctx *Context) error {
	var doLayout bool = true
	var name = ctx.T(template.Layout).OrString(DefaultLayoutName)
	var lext string = typeutil.OrString(filepath.Ext(name), `.html`)

	switch strings.ToLower(name) {
	case `none`, `false`:
		doLayout = false
	case ``:
		doLayout = !strings.HasPrefix(ctx.RequestBasename(), `_`)
	}

	if doLayout {
		var layoutName = template.layoutName(name)
		var layoutPath = filepath.Join(
			typeutil.OrString(ctx.Server().Paths.LayoutsDir, DefaultLayoutsDir),
			name+lext,
		)

		if ctx.WasTemplateSeen(layoutName) {
			return nil
		} else if layoutFile, err := ctx.Open(layoutPath); err == nil {
			defer layoutFile.Close()
			ctx.MarkTemplateSeen(layoutName)

			if err := template.attachTemplate(ctx, layoutName, layoutFile); err == nil {
				template.EntryPoint = layoutName
			} else {
				return err
			}
		} else if name != DefaultLayoutName {
			return err
		}
	}

	if len(template.Includes) > 0 {
		for _, includePath := range template.Includes {
			var includeFullPath = filepath.Join(DefaultIncludesDir, includePath)
			var includeSlug = strings.TrimSuffix(includePath, filepath.Ext(includePath))

			includeSlug = stringutil.Hyphenate(includeSlug)

			if includeFile, err := ctx.Open(includeFullPath); err == nil {
				defer includeFile.Close()
				ctx.MarkTemplateSeen(includeSlug)

				if err := template.attachTemplate(ctx, includeSlug, includeFile); err != nil {
					return errors.Wrapf(err, "include %q (path: %q)", includeSlug, includeFullPath)
				}
			} else {
				return errors.Wrapf(err, "include %q", includeFullPath)
			}
		}
	}

	return nil
}
