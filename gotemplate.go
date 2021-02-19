package diecast

import (
	"io"
	"io/fs"

	htemplate "html/template"
	ttemplate "text/template"
	"text/template/parse"
)

type FuncMap map[string]interface{}

type goTemplate struct {
	html *htemplate.Template
	text *ttemplate.Template
}

func parseGoTemplate(name string, engine string, data string) (*goTemplate, error) {
	var gotmpl = new(goTemplate)

	switch engine {
	case `html`, ``:
		if tmpl, err := htemplate.New(name).Parse(data); err == nil {
			gotmpl.html = tmpl
		} else {
			return nil, err
		}
	case `text`:
		if tmpl, err := ttemplate.New(name).Parse(data); err == nil {
			gotmpl.text = tmpl
		} else {
			return nil, err
		}
	}

	return gotmpl, nil
}

func (self *goTemplate) AddParseTree(name string, tree *parse.Tree) (*goTemplate, error) {
	if self.html != nil {
		var _, err = self.html.AddParseTree(name, tree)
		return self, err
	} else {
		var _, err = self.text.AddParseTree(name, tree)
		return self, err
	}
}

func (self *goTemplate) Clone() (*goTemplate, error) {
	if self.html != nil {
		var _, err = self.html.Clone()
		return self, err
	} else {
		var _, err = self.text.Clone()
		return self, err
	}
}

func (self *goTemplate) DefinedgoTemplates() string {
	if self.html != nil {
		return self.html.DefinedTemplates()
	} else {
		return self.text.DefinedTemplates()
	}
}

func (self *goTemplate) Delims(left, right string) *goTemplate {
	if self.html != nil {
		self.html.Delims(left, right)
	} else {
		self.text.Delims(left, right)
	}

	return self
}

func (self *goTemplate) Execute(wr io.Writer, data interface{}) error {
	if self.html != nil {
		return self.html.Execute(wr, data)
	} else {
		return self.text.Execute(wr, data)
	}
}

func (self *goTemplate) ExecutegoTemplate(wr io.Writer, name string, data interface{}) error {
	if self.html != nil {
		return self.html.ExecuteTemplate(wr, name, data)
	} else {
		return self.text.ExecuteTemplate(wr, name, data)
	}
}

func (self *goTemplate) Funcs(funcMap FuncMap) *goTemplate {
	if self.html != nil {
		var fm = make(htemplate.FuncMap)

		for k, v := range funcMap {
			fm[k] = v
		}

		self.html.Funcs(fm)
	} else {
		var fm = make(ttemplate.FuncMap)

		for k, v := range funcMap {
			fm[k] = v
		}

		self.text.Funcs(fm)
	}

	return self
}

func (self *goTemplate) Lookup(name string) *goTemplate {
	if self.html != nil {
		self.html.Lookup(name)
	} else {
		self.text.Lookup(name)
	}

	return self
}

func (self *goTemplate) Name() string {
	if self.html != nil {
		return self.html.Name()
	} else {
		return self.text.Name()
	}
}
func (self *goTemplate) New(name string) *goTemplate {
	if self.html != nil {
		self.html.New(name)
	} else {
		self.text.New(name)
	}

	return self
}

func (self *goTemplate) Option(opt ...string) *goTemplate {
	if self.html != nil {
		self.html.Option(opt...)
	} else {
		self.text.Option(opt...)
	}

	return self
}

func (self *goTemplate) Parse(text string) (*goTemplate, error) {
	if self.html != nil {
		var _, err = self.html.Parse(text)
		return self, err
	} else {
		var _, err = self.text.Parse(text)
		return self, err
	}
}

func (self *goTemplate) ParseFS(fs fs.FS, patterns ...string) (*goTemplate, error) {
	if self.html != nil {
		var _, err = self.html.ParseFS(fs, patterns...)
		return self, err
	} else {
		var _, err = self.text.ParseFS(fs, patterns...)
		return self, err
	}
}

func (self *goTemplate) ParseFiles(filenames ...string) (*goTemplate, error) {
	if self.html != nil {
		var _, err = self.html.ParseFiles(filenames...)
		return self, err
	} else {
		var _, err = self.text.ParseFiles(filenames...)
		return self, err
	}
}

func (self *goTemplate) ParseGlob(pattern string) (*goTemplate, error) {
	if self.html != nil {
		var _, err = self.html.ParseGlob(pattern)
		return self, err
	} else {
		var _, err = self.text.ParseGlob(pattern)
		return self, err
	}
}
