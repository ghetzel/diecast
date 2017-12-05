package diecast

import (
	"fmt"
	html "html/template"
	"io"
	"path"
	text "text/template"
)

type Engine int

const (
	TextEngine Engine = iota
	HtmlEngine
)

func (self Engine) String() string {
	switch self {
	case TextEngine:
		return `text`
	case HtmlEngine:
		return `html`
	default:
		return `unknown`
	}
}

type FuncMap map[string]interface{}

type Templated interface {
	Parse(text string) error
	Funcs(FuncMap)
	Render(io.Writer, interface{}, string) error
}

type Template struct {
	name   string
	engine Engine
	tmpl   interface{}
	funcs  FuncMap
}

func GetEngineForFile(filename string) Engine {
	switch path.Ext(filename) {
	case `.html`:
		return HtmlEngine
	default:
		return TextEngine
	}
}

func NewTemplate(name string, engine Engine) *Template {
	return &Template{
		name:   name,
		engine: engine,
	}
}

func (self *Template) SetEngine(engine Engine) {
	self.engine = engine
}

func (self *Template) Engine() Engine {
	return self.engine
}

func (self *Template) Parse(input string) error {
	switch self.engine {
	case TextEngine:
		tmpl := text.New(self.name)

		if self.funcs != nil {
			tmpl.Funcs(text.FuncMap(self.funcs))
		}

		if t, err := tmpl.Parse(input); err == nil {
			self.tmpl = t
		} else {
			return err
		}

	case HtmlEngine:
		tmpl := html.New(self.name)

		if self.funcs != nil {
			tmpl.Funcs(html.FuncMap(self.funcs))
		}

		if t, err := tmpl.Parse(input); err == nil {
			self.tmpl = t
		} else {
			return err
		}

	default:
		return fmt.Errorf("Unknown template engine")
	}

	return nil
}

func (self *Template) Funcs(funcs FuncMap) {
	self.funcs = funcs
}

func (self *Template) Render(w io.Writer, data interface{}, subtemplate string) error {
	if self.tmpl == nil {
		return fmt.Errorf("No template input provided")
	}

	var err error

	switch self.engine {
	case TextEngine:
		if t, ok := self.tmpl.(*text.Template); ok {
			if subtemplate == `` {
				err = t.Execute(w, data)
			} else {
				err = t.ExecuteTemplate(w, subtemplate, data)
			}
		} else {
			err = fmt.Errorf("invalid internal type for TextEngine")
		}

	case HtmlEngine:
		if t, ok := self.tmpl.(*html.Template); ok {
			if subtemplate == `` {
				err = t.Execute(w, data)
			} else {
				err = t.ExecuteTemplate(w, subtemplate, data)
			}
		} else {
			err = fmt.Errorf("invalid internal type for HtmlEngine")
		}

	default:
		err = fmt.Errorf("Unknown template engine")
	}

	if err == nil {
		return nil
	} else if terr, ok := err.(*text.ExecError); ok {
		return fmt.Errorf(
			"template %s: %v",
			self.name,
			terr.Err,
		)
	} else if herr, ok := err.(*html.Error); ok {
		return fmt.Errorf(
			"template %s: %v at line %d: %v",
			self.name,
			herr.ErrorCode,
			herr.Line,
			herr.Description,
		)
	} else {
		return fmt.Errorf("template %s: %v", self.name, err)
	}
}
