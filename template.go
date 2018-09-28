package diecast

import (
	"bytes"
	"fmt"
	html "html/template"
	"io"
	"io/ioutil"
	"path"
	"strings"
	text "text/template"

	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/rxutil"
	"github.com/ghetzel/go-stockutil/stringutil"
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
	name           string
	engine         Engine
	tmpl           interface{}
	funcs          FuncMap
	headerOffset   int64
	contentOffset  int64
	postprocessors []PostprocessorFunc
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

func (self *Template) SetHeaderOffset(offset int) {
	self.headerOffset = int64(offset)
}

func (self *Template) AddPostProcessors(postprocessors ...string) error {
	for _, name := range postprocessors {
		if postprocessor, ok := registeredPostprocessors[name]; ok {
			self.postprocessors = append(self.postprocessors, postprocessor)
		} else {
			return fmt.Errorf("No such postprocessor '%v'", name)
		}
	}

	return nil
}

func (self *Template) SetEngine(engine Engine) {
	self.engine = engine
}

func (self *Template) Engine() Engine {
	return self.engine
}

func (self *Template) ParseFrom(r io.Reader) error {
	if data, err := ioutil.ReadAll(r); err == nil {
		return self.Parse(string(data))
	} else {
		return err
	}
}

func (self *Template) Parse(input string) error {
	// determine the line that the "content" template starts on
	for i, line := range strings.Split(input, "\n") {
		if i > 0 && strings.Contains(line, `{{ define "content" }}`) {
			self.contentOffset = int64(i + 2)
			break
		}
	}

	if self.contentOffset > 0 {
		log.Debugf("Template parsed: content offset is %d lines", self.contentOffset)
	}

	switch self.engine {
	case TextEngine:
		tmpl := text.New(self.name)

		if self.funcs != nil {
			tmpl.Funcs(text.FuncMap(self.funcs))
		}

		if t, err := tmpl.Parse(input); err == nil {
			self.tmpl = t
		} else {
			return self.prepareError(err)
		}

	case HtmlEngine:
		tmpl := html.New(self.name)

		if self.funcs != nil {
			tmpl.Funcs(html.FuncMap(self.funcs))
		}

		if t, err := tmpl.Parse(input); err == nil {
			self.tmpl = t
		} else {
			return self.prepareError(err)
		}

	default:
		return fmt.Errorf("Unknown template engine")
	}

	return self.preprocessTemplate()
}

func (self *Template) preprocessTemplate() error {

	return nil
}

func (self *Template) Funcs(funcs FuncMap) {
	self.funcs = funcs
}

func (self *Template) prepareError(err error) error {
	if err == nil {
		return nil
	} else {
		msg := err.Error()

		// get the filename to look like a relative path
		if match := rxutil.Match(`^template: ([^:]+)`, msg); match != nil {
			msg = match.ReplaceGroup(
				1,
				strings.TrimPrefix(strings.Replace(match.Group(1), `-`, `/`, -1), `/`),
			)
		}

		// adjust the line number to match the file by accounting for offsets
		if match := rxutil.Match(`(?:line|:)(\d+)`, msg); match != nil {
			if v := match.Group(1); v != `` {
				if vI, err := stringutil.ConvertToInteger(v); err == nil {
					if vI > self.contentOffset {
						vI = (vI - self.contentOffset) + self.headerOffset
						msg = match.ReplaceGroup(1, fmt.Sprintf("%v", vI))
					}
				}
			}
		}

		// prettify the sentence a little
		if match := rxutil.Match(`^template: [^:]+(:\d+)`, msg); match != nil {
			msg = match.ReplaceGroup(
				1,
				fmt.Sprintf(", line %s", strings.TrimPrefix(match.Group(1), `:`)),
			)

			msg = fmt.Sprintf("Error in %v", strings.TrimPrefix(msg, `template: `))
		}

		return fmt.Errorf("%v", msg)
	}
}

func (self *Template) Render(w io.Writer, data interface{}, subtemplate string) error {
	if self.tmpl == nil {
		return fmt.Errorf("No template input provided")
	}

	output := bytes.NewBuffer(nil)
	var err error

	switch self.engine {
	case TextEngine:
		if t, ok := self.tmpl.(*text.Template); ok {
			if subtemplate == `` {
				err = t.Execute(output, data)
			} else {
				err = t.ExecuteTemplate(output, subtemplate, data)
			}
		} else {
			err = fmt.Errorf("invalid internal type for TextEngine")
		}

	case HtmlEngine:
		if t, ok := self.tmpl.(*html.Template); ok {
			if subtemplate == `` {
				err = t.Execute(output, data)
			} else {
				err = t.ExecuteTemplate(output, subtemplate, data)
			}
		} else {
			err = fmt.Errorf("invalid internal type for HtmlEngine")
		}

	default:
		err = fmt.Errorf("Unknown template engine")
	}

	if err == nil {
		outstr := output.String()

		for n, postprocessor := range self.postprocessors {
			if out, err := postprocessor(outstr); err == nil {
				outstr = out
			} else {
				return self.prepareError(
					fmt.Errorf("Postprocessor %d: %v", n, err),
				)
			}
		}

		_, err = w.Write([]byte(outstr))
	}

	return self.prepareError(err)
}
