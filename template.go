package diecast

import (
	"bytes"
	"fmt"
	html "html/template"
	"io"
	"net/http"
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

func (tpl Engine) String() string {
	switch tpl {
	case TextEngine:
		return `text`
	case HtmlEngine:
		return `html`
	default:
		return `unknown`
	}
}

type FuncMap map[string]any

type Template struct {
	name           string
	engine         Engine
	tmpl           any
	funcs          FuncMap
	headerOffset   int64
	contentOffset  int64
	postprocessors []PostprocessorFunc
	delimOpen      string
	delimClose     string
	prewrite       func()
}

func GetEngineForFile(filename string) Engine {
	switch strings.ToLower(path.Ext(filename)) {
	case `.html`, `.htm`:
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

func (tpl *Template) SetPrewriteFunc(fn func()) {
	tpl.prewrite = fn
}

func (tpl *Template) SetHeaderOffset(offset int) {
	tpl.headerOffset = int64(offset)
}

func (tpl *Template) SetDelimiters(open string, close string) {
	tpl.delimOpen = open
	tpl.delimClose = close
}

func (tpl *Template) AddPostProcessors(postprocessors ...string) error {
	for _, name := range postprocessors {
		if postprocessor, ok := registeredPostprocessors[name]; ok {
			tpl.postprocessors = append(tpl.postprocessors, postprocessor)
		} else {
			return fmt.Errorf("no such postprocessor '%v'", name)
		}
	}

	return nil
}

func (tpl *Template) SetEngine(engine Engine) {
	tpl.engine = engine
}

func (tpl *Template) Engine() Engine {
	return tpl.engine
}

func (tpl *Template) ParseFrom(r io.Reader) error {
	if data, err := io.ReadAll(r); err == nil {
		return tpl.ParseString(string(data))
	} else {
		return err
	}
}

func (tpl *Template) ParseString(input string) error {
	// determine the line that the "content" template starts on
	for i, line := range strings.Split(input, "\n") {
		if i > 0 && strings.Contains(line, `{{ define "content" }}`) {
			tpl.contentOffset = int64(i + 2)
			break
		}
	}

	if tpl.contentOffset > 0 {
		log.Debugf("Template parsed: content offset is %d lines", tpl.contentOffset)
	}

	switch tpl.engine {
	case TextEngine:
		var tmpl = text.New(tpl.name)

		if tpl.funcs != nil {
			tmpl.Funcs(text.FuncMap(tpl.funcs))
		}

		if t, err := tmpl.Parse(input); err == nil {
			tpl.tmpl = t
		} else {
			return tpl.prepareError(err)
		}

	case HtmlEngine:
		var tmpl = html.New(tpl.name)

		if tpl.funcs != nil {
			tmpl.Funcs(html.FuncMap(tpl.funcs))
		}

		if t, err := tmpl.Parse(input); err == nil {
			tpl.tmpl = t
		} else {
			return tpl.prepareError(err)
		}

	default:
		return fmt.Errorf("unknown template engine")
	}

	return nil
}

func (tpl *Template) ParseFragments(fragments FragmentSet) error {
	var hasLayout = fragments.HasLayout()

	switch tpl.engine {
	case TextEngine:
		var tmpl = text.New(tpl.name)

		if tpl.funcs != nil {
			tmpl.Funcs(text.FuncMap(tpl.funcs))
		}

		for _, fragment := range fragments {
			var t *text.Template

			if !hasLayout && fragment.Name == ContentTemplateName {
				t = tmpl
			} else {
				t = tmpl.New(fragment.Name)
			}

			if _, err := t.Parse(string(fragment.Data)); err != nil {
				return fmt.Errorf("textEngine: error parsing fragment %q: %v", fragment.Name, err)
			}
		}

		tpl.tmpl = tmpl

	case HtmlEngine:
		var tmpl = html.New(tpl.name)

		if tpl.funcs != nil {
			tmpl.Funcs(html.FuncMap(tpl.funcs))
		}

		for _, fragment := range fragments {
			var t *html.Template

			if !hasLayout && fragment.Name == ContentTemplateName {
				t = tmpl
			} else {
				t = tmpl.New(fragment.Name)
			}

			if _, err := t.Parse(string(fragment.Data)); err != nil {
				return fmt.Errorf("HtmlEngine: error parsing fragment %q: %v", fragment.Name, err)
			}
		}

		tpl.tmpl = tmpl

	default:
		return fmt.Errorf("unknown template engine")
	}

	return nil
}

func (tpl *Template) Funcs(funcs FuncMap) {
	tpl.funcs = funcs
}

func (tpl *Template) prepareError(err error) error {
	if err == nil {
		return nil
	} else {
		var msg = err.Error()

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
					if vI > tpl.contentOffset {
						vI = (vI - tpl.contentOffset) + tpl.headerOffset
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

			msg = fmt.Sprintf("error in %v", strings.TrimPrefix(msg, `template: `))
		}

		return fmt.Errorf("%v", msg)
	}
}

func (tpl *Template) Render(w io.Writer, data any, subtemplate string) error {
	return tpl.renderWithRequest(nil, w, data, subtemplate)
}

func (tpl *Template) renderWithRequest(req *http.Request, w io.Writer, data any, subtemplate string) error {
	if tpl.tmpl == nil {
		return fmt.Errorf("no template input provided")
	}

	var output = bytes.NewBuffer(nil)
	var err error

	switch tpl.engine {
	case TextEngine:
		if t, ok := tpl.tmpl.(*text.Template); ok {
			t.Delims(tpl.delimOpen, tpl.delimClose)

			if subtemplate == `` {
				err = t.Execute(output, data)
			} else {
				err = t.ExecuteTemplate(output, subtemplate, data)
			}
		} else {
			err = fmt.Errorf("invalid internal type for TextEngine")
		}

	case HtmlEngine:
		if t, ok := tpl.tmpl.(*html.Template); ok {
			t.Delims(tpl.delimOpen, tpl.delimClose)

			if subtemplate == `` {
				err = t.Execute(output, data)
			} else {
				err = t.ExecuteTemplate(output, subtemplate, data)
			}
		} else {
			err = fmt.Errorf("invalid internal type for HtmlEngine")
		}

	default:
		err = fmt.Errorf("unknown template engine")
	}

	if err == nil {
		var outstr = output.String()

		for n, postprocessor := range tpl.postprocessors {
			if out, err := postprocessor(outstr, req); err == nil {
				outstr = out
			} else {
				return tpl.prepareError(
					fmt.Errorf("postprocessor %d: %v", n, err),
				)
			}
		}

		if fn := tpl.prewrite; fn != nil {
			fn()
		}

		_, werr := w.Write([]byte(outstr))
		err = werr
	}

	return tpl.prepareError(err)
}
