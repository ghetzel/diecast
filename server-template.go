package diecast

import (
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"path"
	"path/filepath"
)

func (self *Server) RenderTemplateFromRequest(name string, reader io.Reader, w http.ResponseWriter, req *http.Request) (bool, error) {
	if tmpl, err := self.LoadTemplate(name, reader); err == nil {
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

		log.Debugf("Rendering %q as template", tmpl.Name())

		if err := tmpl.Execute(w, payload); err != nil {
			return true, err
		}

		return true, nil
	} else {
		return false, err
	}
}

func (self *Server) LoadTemplate(name string, reader io.Reader) (*template.Template, error) {
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
		return nil, fmt.Errorf("Request path %q is not a candidate for templating", name)
	}

	if data, err := ioutil.ReadAll(reader); err == nil {
		return template.New(name).Parse(string(data[:]))
	} else {
		return nil, err
	}
}
