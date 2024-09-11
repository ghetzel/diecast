package internal

import (
	"io"
	"io/fs"

	htemplate "html/template"
	ttemplate "text/template"
	"text/template/parse"
)

type FuncMap map[string]interface{}

const TextEngine string = `text`
const HtmlEngine string = `html`

func NewTemplate(name string, engine string) *GolangTemplate {
	var t, _ = ParseGolangTemplate(name, engine, ``, BuiltinFunctions)
	return t
}

func NewTemplateWithFuncs(name string, engine string, funcs FuncMap) *GolangTemplate {
	var t, _ = ParseGolangTemplate(name, engine, ``, funcs)
	return t
}

type GolangTemplate struct {
	html   *htemplate.Template
	text   *ttemplate.Template
	name   string
	engine string
	body   string
	funcs  FuncMap
}

func ParseGolangTemplate(name string, engine string, body string, funcs FuncMap) (*GolangTemplate, error) {
	var gotmpl = new(GolangTemplate)

	gotmpl.name = name
	gotmpl.engine = engine
	gotmpl.body = body
	gotmpl.funcs = funcs

	return gotmpl, gotmpl.init()
}

func (self *GolangTemplate) init() error {
	switch self.engine {
	case HtmlEngine, ``:
		if tmpl, err := htemplate.New(self.name).Funcs(
			self.hfuncs(),
		).Parse(self.body); err == nil {
			self.html = tmpl
			self.text = nil
		} else {
			return err
		}
	case TextEngine:
		if tmpl, err := ttemplate.New(self.name).Funcs(
			self.tfuncs(),
		).Parse(self.body); err == nil {
			self.html = nil
			self.text = tmpl
		} else {
			return err
		}
	}

	return nil
}

func (self *GolangTemplate) Names() (names []string) {
	if self.html != nil {
		for _, t := range self.html.Templates() {
			names = append(names, t.Name())
		}
	} else {
		for _, t := range self.text.Templates() {
			names = append(names, t.Name())
		}
	}

	return
}

func (self *GolangTemplate) ParseTree() *parse.Tree {
	if self.html != nil {
		return self.html.Tree.Copy()
	} else {
		return self.text.Tree.Copy()
	}
}

func (self *GolangTemplate) AddParseTree(name string, tree *parse.Tree) (*GolangTemplate, error) {
	if self.html != nil {
		var _, err = self.html.AddParseTree(name, tree)
		return self, err
	} else {
		var _, err = self.text.AddParseTree(name, tree)
		return self, err
	}
}

func (self *GolangTemplate) Clone() (*GolangTemplate, error) {
	if self.html != nil {
		var _, err = self.html.Clone()
		return self, err
	} else {
		var _, err = self.text.Clone()
		return self, err
	}
}

func (self *GolangTemplate) DefinedTemplates() string {
	if self.html != nil {
		return self.html.DefinedTemplates()
	} else {
		return self.text.DefinedTemplates()
	}
}

func (self *GolangTemplate) Delims(left, right string) *GolangTemplate {
	if self.html != nil {
		self.html.Delims(left, right)
	} else {
		self.text.Delims(left, right)
	}

	return self
}

func (self *GolangTemplate) Execute(wr io.Writer, data interface{}) error {
	if self.html != nil {
		return self.html.Execute(wr, data)
	} else {
		return self.text.Execute(wr, data)
	}
}

func (self *GolangTemplate) ExecuteTemplate(wr io.Writer, name string, data interface{}) error {
	if self.html != nil {
		return self.html.ExecuteTemplate(wr, name, data)
	} else {
		return self.text.ExecuteTemplate(wr, name, data)
	}
}

// Backwards compat with 1.x
func (self *GolangTemplate) Render(wr io.Writer, data interface{}, name string) error {
	if self.html != nil {
		return self.html.ExecuteTemplate(wr, name, data)
	} else {
		return self.text.ExecuteTemplate(wr, name, data)
	}
}

func (self *GolangTemplate) hfuncs() htemplate.FuncMap {
	var fm = make(htemplate.FuncMap)

	for k, v := range self.funcs {
		fm[k] = v
	}

	return fm
}

func (self *GolangTemplate) tfuncs() ttemplate.FuncMap {
	var fm = make(ttemplate.FuncMap)

	for k, v := range self.funcs {
		fm[k] = v
	}

	return fm
}

func (self *GolangTemplate) Lookup(name string) *GolangTemplate {
	if self.html != nil {
		self.html.Lookup(name)
	} else {
		self.text.Lookup(name)
	}

	return self
}

func (self *GolangTemplate) Name() string {
	if self.html != nil {
		return self.html.Name()
	} else {
		return self.text.Name()
	}
}
func (self *GolangTemplate) New(name string) *GolangTemplate {
	if self.html != nil {
		self.html.New(name)
	} else {
		self.text.New(name)
	}

	return self
}

func (self *GolangTemplate) Option(opt ...string) *GolangTemplate {
	if self.html != nil {
		self.html.Option(opt...)
	} else {
		self.text.Option(opt...)
	}

	return self
}

func (self *GolangTemplate) Parse(text string) (*GolangTemplate, error) {
	if self.html != nil {
		var _, err = self.html.Parse(text)
		return self, err
	} else {
		var _, err = self.text.Parse(text)
		return self, err
	}
}

func (self *GolangTemplate) ParseFS(fs fs.FS, patterns ...string) (*GolangTemplate, error) {
	if self.html != nil {
		var _, err = self.html.ParseFS(fs, patterns...)
		return self, err
	} else {
		var _, err = self.text.ParseFS(fs, patterns...)
		return self, err
	}
}

func (self *GolangTemplate) ParseFiles(filenames ...string) (*GolangTemplate, error) {
	if self.html != nil {
		var _, err = self.html.ParseFiles(filenames...)
		return self, err
	} else {
		var _, err = self.text.ParseFiles(filenames...)
		return self, err
	}
}

func (self *GolangTemplate) ParseGlob(pattern string) (*GolangTemplate, error) {
	if self.html != nil {
		var _, err = self.html.ParseGlob(pattern)
		return self, err
	} else {
		var _, err = self.text.ParseGlob(pattern)
		return self, err
	}
}

func (self *GolangTemplate) ParseString(body string) error {
	self.body = body
	return self.init()
}
