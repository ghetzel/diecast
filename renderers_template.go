package diecast

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghodss/yaml"
)

type TemplateRenderer struct {
	server   *Server
	prewrite PrewriteFunc
}

func (self *TemplateRenderer) ShouldPrerender() bool {
	return false
}

func (self *TemplateRenderer) SetServer(server *Server) {
	self.server = server
}

func (self *TemplateRenderer) SetPrewriteFunc(fn PrewriteFunc) {
	self.prewrite = fn
}

func (self *TemplateRenderer) Render(w http.ResponseWriter, req *http.Request, options RenderOptions) error {
	if len(options.Fragments) == 0 {
		return fmt.Errorf("Must specify a non-empty FragmentSet to TemplateRenderer")
	}

	// create the template and make it aware of our custom functions
	tmpl := NewTemplate(
		self.server.ToTemplateName(options.RequestedPath),
		GetEngineForFile(options.RequestedPath),
	)

	if fn := self.prewrite; fn != nil {
		tmpl.SetPrewriteFunc(func() {
			fn(req)
		})
	}

	tmpl.Funcs(options.FunctionSet)
	tmpl.SetHeaderOffset(options.HeaderOffset)

	// if delim := options.Header.Delimiters; len(delim) == 2 {
	// 	tmpl.SetDelimiters(delim[0], delim[1])
	// }

	if err := tmpl.AddPostProcessors(options.Header.Postprocessors...); err != nil {
		return err
	}

	if err := tmpl.ParseFragments(options.Fragments); err == nil {
		log.Debugf("[%s] Rendering %q as %v template", reqid(req), options.RequestedPath, tmpl.Engine())

		if hdr := options.Header; hdr != nil {
			// include any configured response headers now
			for name, value := range hdr.Headers {
				w.Header().Set(name, MustEvalInline(
					fmt.Sprintf("%v", value),
					options.Data,
					options.FunctionSet,
				))
			}

			if hdr.StatusCode > 0 {
				w.WriteHeader(hdr.StatusCode)
			}
		}

		if options.MimeType == `` {
			options.MimeType = `text/html; charset=utf-8`
		}

		// this entire if-block is just for debugging templates
		if self.server.ShouldReturnSource(req) {
			w.Header().Set(`Content-Type`, `text/plain`)

			if fn := self.prewrite; fn != nil {
				fn(req)
			}

			if hdr, err := yaml.Marshal(options.Header); err == nil {
				w.Write([]byte("{{/* BEGIN COMBINED HEADER --\n"))
				w.Write(hdr)
				w.Write([]byte("\n-- END COMBINED HEADER */}}\n"))
			} else {
				w.Write([]byte(fmt.Sprintf("{{/* COMBINED HEADER: error: %v */}}\n", err)))
			}

			dV := options.Data
			delete(dV, `bindings`)

			if data, err := yaml.Marshal(dV); err == nil {
				w.Write([]byte("{{/* BEGIN DATA --\n"))
				w.Write(data)
				w.Write([]byte("\n-- END DATA */}}\n"))
			} else {
				w.Write([]byte(fmt.Sprintf("{{/* DATA: error: %v */}}\n", err)))
			}

			if _, err := w.Write(options.Fragments.DebugOutput()); err != nil {
				return err
			}

			return nil
		} else {
			w.Header().Set(`Content-Type`, options.MimeType)

			if options.Fragments.HasLayout() {
				return tmpl.Render(w, options.Data, LayoutTemplateName)
			} else {
				return tmpl.Render(w, options.Data, ``)
			}
		}
	} else if self.server.ShouldReturnSource(req) {
		var tplstr string
		lines := strings.Split(string(options.Fragments.DebugOutput()), "\n")
		lineNoSpaces := fmt.Sprintf("%d", len(fmt.Sprintf("%d", len(lines)))+1)

		for i, line := range lines {
			tplstr += fmt.Sprintf("% "+lineNoSpaces+"d | %s\n", i+1, line)
		}

		tplstr = fmt.Sprintf("ERROR: %v\n\n", err) + tplstr

		w.Header().Set(`Content-Type`, `text/plain; charset=utf-8`)

		if fn := self.prewrite; fn != nil {
			fn(req)
		}

		w.Write([]byte(tplstr))

		return nil
	} else {
		return err
	}
}
