package diecast

import (
	"bytes"
	"fmt"
	html "html/template"
	"io"
	"io/ioutil"
	"net/http"
	"path"
	"strings"
	text "text/template"
	"text/template/parse"

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

type Template struct {
	name           string
	engine         Engine
	tmpl           interface{}
	funcs          FuncMap
	headerOffset   int64
	contentOffset  int64
	postprocessors []PostprocessorFunc
	delimOpen      string
	delimClose     string
	prewrite       func()
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

func (self *Template) SetPrewriteFunc(fn func()) {
	self.prewrite = fn
}

func (self *Template) SetHeaderOffset(offset int) {
	self.headerOffset = int64(offset)
}

func (self *Template) SetDelimiters(open string, close string) {
	self.delimOpen = open
	self.delimClose = close
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
		return self.ParseString(string(data))
	} else {
		return err
	}
}

func (self *Template) ParseString(input string) error {
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
		var tmpl = text.New(self.name)

		if self.funcs != nil {
			tmpl.Funcs(text.FuncMap(self.funcs))
		}

		if t, err := tmpl.Parse(input); err == nil {
			self.tmpl = t
		} else {
			return self.prepareError(err)
		}

	case HtmlEngine:
		var tmpl = html.New(self.name)

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

	return nil
}

func (self *Template) ParseFragments(fragments FragmentSet) error {
	var hasLayout = fragments.HasLayout()

	switch self.engine {
	case TextEngine:
		var tmpl = text.New(self.name)

		if self.funcs != nil {
			tmpl.Funcs(text.FuncMap(self.funcs))
		}

		for _, fragment := range fragments {
			var t *text.Template

			if !hasLayout && fragment.Name == ContentTemplateName {
				t = tmpl
			} else {
				t = tmpl.New(fragment.Name)
			}

			if _, err := t.Parse(string(fragment.Data)); err != nil {
				return fmt.Errorf("error parsing fragment %s: %v", fragment.Name, err)
			}
		}

		self.tmpl = tmpl

	case HtmlEngine:
		var tmpl = html.New(self.name)

		if self.funcs != nil {
			tmpl.Funcs(html.FuncMap(self.funcs))
		}

		for _, fragment := range fragments {
			var t *html.Template

			if !hasLayout && fragment.Name == ContentTemplateName {
				t = tmpl
			} else {
				t = tmpl.New(fragment.Name)
			}

			if _, err := t.Parse(string(fragment.Data)); err != nil {
				return fmt.Errorf("error parsing fragment %s: %v", fragment.Name, err)
			}
		}

		self.tmpl = tmpl

	default:
		return fmt.Errorf("Unknown template engine")
	}

	return nil
}

func (self *Template) Funcs(funcs FuncMap) {
	self.funcs = funcs
}

func (self *Template) prepareError(err error) error {
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
	return self.renderWithRequest(nil, w, data, subtemplate)
}

func (self *Template) renderWithRequest(req *http.Request, w io.Writer, data interface{}, subtemplate string) error {
	if self.tmpl == nil {
		return fmt.Errorf("No template input provided")
	}

	var output = bytes.NewBuffer(nil)
	var err error

	switch self.engine {
	case TextEngine:
		if t, ok := self.tmpl.(*text.Template); ok {
			t.Delims(self.delimOpen, self.delimClose)

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
			t.Delims(self.delimOpen, self.delimClose)

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
		var outstr = output.String()

		for n, postprocessor := range self.postprocessors {
			if out, err := postprocessor(outstr, req); err == nil {
				outstr = out
			} else {
				return self.prepareError(
					fmt.Errorf("Postprocessor %d: %v", n, err),
				)
			}
		}

		if fn := self.prewrite; fn != nil {
			fn()
		}

		_, werr := w.Write([]byte(outstr))
		err = werr
	}

	return self.prepareError(err)
}

func (self *Template) prepareParseTree(tree *parse.Tree) error {
	// log.Debug("ROOT:")

	// for _, node := range tree.Root.Nodes {
	// 	self.prepareNode(node, 1)
	// }

	return nil
}

func (self *Template) prepareNode(tree *parse.Tree, node parse.Node, depth int) {
	var repr string

	log.Debugf("%v%T", strings.Repeat(`  `, depth), node)

	switch node.(type) {
	case *parse.RangeNode:
		self.prepareNode(tree, node.(*parse.RangeNode).Pipe, depth+1)
	case *parse.PipeNode:
		for _, decl := range node.(*parse.PipeNode).Decl {
			self.prepareNode(tree, decl, depth+1)
		}

		for _, cmd := range node.(*parse.PipeNode).Cmds {
			self.prepareNode(tree, cmd, depth+1)
		}
	case *parse.VariableNode:
		var varnode = node.(*parse.VariableNode)
		repr = node.(*parse.VariableNode).String()
		var idents = varnode.Ident

		for i, ident := range idents {
			log.Debugf("%v%d: %v", strings.Repeat(`  `, depth+1), i, ident)
		}

		// if len(idents) > 1 {
		// 	replace := parse.NewIdentifier(`get`).SetPos(node.Position()).SetTree(tree)
		// }

	case *parse.CommandNode:
		repr = node.(*parse.CommandNode).String()

		for _, arg := range node.(*parse.CommandNode).Args {
			self.prepareNode(tree, arg, depth+1)
		}

	case *parse.IdentifierNode:
		log.Debugf("%v: %v", strings.Repeat(`  `, depth+1), node.(*parse.IdentifierNode).Ident)
	}

	if repr != `` {
		log.Debugf("%v%s", strings.Repeat(`  `, depth), repr)
	}
}
