package diecast

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
)

func (self *Server) RenderTemplateFromRequest(name string, reader io.Reader, w http.ResponseWriter, req *http.Request) (bool, error) {
	payload := map[string]interface{}{
		`request`: req,
		`server`:  self,
	}

	for _, binding := range self.Config.Bindings {
		if binding.ShouldEvaluate(req) {
			if _, ok := payload[binding.Name]; !ok {
				if value, err := binding.Evaluate(req); err == nil && value != nil {
					log.Debugf("Got results for binding %q", binding.Name)
					payload[binding.Name] = value
				} else {
					return true, fmt.Errorf("Binding %q failed: %v", binding.Name, err)
				}
			} else {
				return true, fmt.Errorf("Binding %q failed: key already exists in payload", binding.Name)
			}
		}
	}

	templateData := bytes.NewBuffer(nil)
	wrappers := make([]string, 0)

	// if a layout template is specified, read that first and write it to a temporary
	// buffer that will hold the entire template chain
	if layout := self.DefaultTemplate; layout != `` {
		layoutName := layout

		if path.Ext(layout) == `` {
			layout = fmt.Sprintf("%s.html", layout)
		}

		layout = path.Join(self.RootPath, `_layouts`, layout)

		log.Debugf("Loading layout template %q from file %q", layoutName, layout)

		if layoutTmplFile, err := os.Open(layout); err == nil {
			if err := self.WriteTemplate(templateData, layoutName, layoutTmplFile); err != nil {
				return true, err
			}

			// because we just loaded and added a layout template, tell all subsequent templates
			// to wrap themselves in a "content" definition
			wrappers = append(wrappers, `content`)
		} else {
			return true, err
		}
	}

	log.Debugf("Rendering %q as template", name)

	// write template data to buffer
	if err := self.WriteTemplate(templateData, name, reader, wrappers...); err == nil {
		// parse template
		if tmpl, err := template.New(name).Parse(templateData.String()); err == nil {
			// render to response writer and return the result
			return true, tmpl.ExecuteTemplate(w, tmpl.Name(), payload)
		} else {
			return true, err
		}
	} else {
		return true, err
	}
}

func (self *Server) WriteTemplate(w io.Writer, name string, reader io.Reader, wrapWithDefs ...string) error {
	shouldExit := true
	name = path.Base(name)

	if self.mountProxy.TemplatePatterns != nil {
		for _, pattern := range self.mountProxy.TemplatePatterns {
			log.Debugf("Check pattern %q against %q", pattern, name)

			if matches, err := filepath.Match(pattern, name); err == nil {
				if matches {
					log.Debugf("File %q matched using pattern %q", name, pattern)
					shouldExit = false
					break
				}
			} else {
				log.Warningf("Invalid template match pattern %q: %v", pattern, err)
			}
		}
	}

	if shouldExit {
		return fmt.Errorf("Request path %q is not a candidate for templating", name)
	}

	if data, err := ioutil.ReadAll(reader); err == nil {
		content := string(data[:])

		for _, wrap := range wrapWithDefs {
			log.Debugf("Wrapping template %q in %q-block", name, wrap)
			content = fmt.Sprintf("{{ define %q }}\n%s\n{{ end }}\n", content)
		}

		_, err := io.Copy(w, bytes.NewBufferString(content))
		return err
	} else {
		return err
	}
}
