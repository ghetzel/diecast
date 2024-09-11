package diecast

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/ghetzel/diecast/v2/internal"
	"github.com/ghetzel/go-stockutil/typeutil"
)

var DefaultEntryPoint = `content`
var DefaultLayoutName = `default`
var DefaultTemplateEngine = `html`
var Delimiters = internal.Delimiters
var FrontMatterSeparator = internal.FrontMatterSeparator
var LayoutNamePrefix string = `layout:`
var MaxFrontMatterSize = 32768

type Template struct {
	*internal.TemplateHeader
	body     []byte
	buf      *bytes.Buffer
	gotmpl   *internal.GolangTemplate
	initDone bool
}

func ParseTemplateString(source string) (*Template, error) {
	return ParseTemplate(bytes.NewBufferString(source))
}

func ParseTemplate(source io.Reader) (*Template, error) {
	if source == nil {
		return nil, io.EOF
	}

	var tmpl = new(Template)

	if hdr, body, err := internal.SplitTemplateHeaderContent(source); err == nil {
		tmpl.TemplateHeader = hdr
		tmpl.body = body
	} else {
		return nil, err
	}

	return tmpl, tmpl.init()
}

// Implement reader interface.
func (self *Template) Read(b []byte) (int, error) {
	if self.buf == nil {
		if err := self.init(); err == nil {
			var dst bytes.Buffer

			if err := self.Render(nil, &dst); err != nil {
				return 0, err
			}

			self.buf = &dst
		} else {
			return 0, fmt.Errorf("init err: %v", err)
		}
	}

	if buf := self.buf; buf != nil {
		return buf.Read(b)
	} else {
		return 0, io.EOF
	}
}

// Implemented io.Closer
func (self *Template) Close() error {
	self.initDone = false
	self.buf = nil
	self.gotmpl = nil
	return nil
}

// Initialize the template, parsing the data and making the object ready for subsequent calls to Render
func (self *Template) init() error {
	if self.initDone {
		return nil
	}

	var engine = typeutil.OrString(self.Engine, DefaultTemplateEngine)
	// var name = typeutil.OrString(self.Filename, engine+`:`+self.SHA512SUM)

	if gotmpl, err := internal.ParseGolangTemplate(self.entryPoint(), engine, self.TemplateString()); err == nil {
		self.gotmpl = gotmpl
		self.initDone = true
		return nil
	} else {
		return err
	}
}

// Return the raw, unrendered template source.
func (self *Template) TemplateString() string {
	return string(self.body)
}

// Implement fmt.Stringer
func (self *Template) String() string {
	if err := self.init(); err == nil {
		var dst bytes.Buffer

		if err := self.Render(nil, &dst); err == nil {
			return dst.String()
		} else {
			return fmt.Sprintf("<!-- TEMPLATE ERROR: %v -->", err)
		}
	} else {
		return ``
	}
}

func (self *Template) entryPoint() string {
	return typeutil.OrString(self.EntryPoint, DefaultEntryPoint)
}

// Refresh all data sources and render the template, writing the results to the giveni io.Writer.
func (self *Template) Render(ctx *Context, w io.Writer) error {
	if err := self.init(); err != nil {
		return err
	}

	if ctx == nil {
		ctx = NewContext(nil)
	}

	if w == nil {
		w = ctx
	}

	// ctx.Debugf("template: known templates: %s", strings.Join(self.gotmpl.Names()``, `, `))
	// ctx.Debugf("template: entrypoint: %s", self.entryPoint())

	return self.gotmpl.ExecuteTemplate(w, self.entryPoint(), ctx.Data())
}

// Returns the SHA512 checksum of the underlying template file.
func (self *Template) Checksum() string {
	return self.SHA512SUM
}

func (self *Template) attachTemplate(ctx *Context, tmplName string, r io.Reader) error {
	if tmpl, err := ParseTemplate(r); err == nil {
		if err := tmpl.LoadRelatedTemplates(ctx); err != nil {
			return fmt.Errorf("%s: %v", tmplName, err)
		}

		// whatever we need to do to merge in the new template header, do it here
		self.EntryPoint = typeutil.OrString(tmpl.EntryPoint, self.EntryPoint)
		self.DataSources = append(tmpl.DataSources, self.DataSources...)

		// add this new data to our existing template tree and return
		if pt := tmpl.gotmpl.ParseTree(); pt != nil {
			var _, err = self.gotmpl.AddParseTree(tmplName, pt)
			return err
		} else {
			return fmt.Errorf("invalid layout template")
		}
	} else {
		return err
	}
}

func (self *Template) layoutName(name string) string {
	return LayoutNamePrefix + strings.TrimPrefix(name, LayoutNamePrefix)
}

func (self *Template) LoadRelatedTemplates(ctx *Context) error {
	var doLayout bool = true
	var name = ctx.T(self.Layout).OrString(DefaultLayoutName)
	var lext string = typeutil.OrString(filepath.Ext(name), `.html`)

	switch strings.ToLower(name) {
	case `none`, `false`:
		doLayout = false
	case ``:
		doLayout = !strings.HasPrefix(ctx.RequestBasename(), `_`)
	}

	if doLayout {
		var layoutName = self.layoutName(name)
		var layoutPath = filepath.Join(
			typeutil.OrString(ctx.Server().Paths.LayoutsDir, DefaultLayoutsDir),
			name+lext,
		)

		if ctx.WasTemplateSeen(layoutName) {
			return nil
		} else if layoutFile, err := ctx.Open(layoutPath); err == nil {
			defer layoutFile.Close()
			ctx.MarkTemplateSeen(layoutName)

			if err := self.attachTemplate(ctx, layoutName, layoutFile); err == nil {
				self.EntryPoint = layoutName
			} else {
				return err
			}
		} else if name != DefaultLayoutName {
			return err
		}
	}

	return nil
}
