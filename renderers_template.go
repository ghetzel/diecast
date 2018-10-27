package diecast

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghodss/yaml"
)

type TemplateRenderer struct {
	server *Server
}

func (self *TemplateRenderer) ShouldPrerender() bool {
	return false
}

func (self *TemplateRenderer) Render(w http.ResponseWriter, req *http.Request, options RenderOptions) error {
	defer options.Input.Close()

	// create the template and make it aware of our custom functions
	tmpl := NewTemplate(
		self.server.ToTemplateName(options.RequestedPath),
		GetEngineForFile(options.RequestedPath),
	)

	tmpl.Funcs(options.FunctionSet)
	tmpl.SetHeaderOffset(options.HeaderOffset)

	if err := tmpl.AddPostProcessors(options.Header.Postprocessors...); err != nil {
		return err
	}

	if data, err := ioutil.ReadAll(options.Input); err == nil {
		if err := tmpl.Parse(string(data)); err == nil {
			log.Debugf("Rendering %q as %v template (header offset by %d lines)", options.RequestedPath, tmpl.Engine(), options.HeaderOffset)

			if options.Header != nil {
				// include any configured response headers now
				for name, value := range options.Header.Headers {
					w.Header().Set(name, fmt.Sprintf("%v", value))
				}
			}

			if self.server.ShouldReturnSource(req) {
				w.Header().Set(`Content-Type`, `text/plain`)

				if hdr, err := yaml.Marshal(options.Header); err == nil {
					w.Write([]byte("{{/* BEGIN COMBINED HEADER --\n"))
					w.Write(hdr)
					w.Write([]byte("\n-- END COMBINED HEADER */}}\n"))
				} else {
					w.Write([]byte(fmt.Sprintf("{{/* COMBINED HEADER: error: %v */}}\n", err)))
				}

				if _, err := w.Write(data); err != nil {
					return err
				}

				return nil
			} else {
				w.Header().Set(`Content-Type`, options.MimeType)

				if options.HasLayout {
					return tmpl.Render(w, options.Data, `layout`)
				} else {
					return tmpl.Render(w, options.Data, ``)
				}
			}
		} else if self.server.ShouldReturnSource(req) {
			var tplstr string
			lines := strings.Split(string(data), "\n")
			lineNoSpaces := fmt.Sprintf("%d", len(fmt.Sprintf("%d", len(lines)))+1)

			for i, line := range lines {
				tplstr += fmt.Sprintf("% "+lineNoSpaces+"d | %s\n", i+1, line)
			}

			tplstr = fmt.Sprintf("ERROR: %v\n\n", err) + tplstr

			w.Header().Set(`Content-Type`, `text/plain; charset=utf-8`)
			w.Write([]byte(tplstr))

			return nil
		} else {
			return err
		}
	} else {
		return err
	}
}
