package diecast

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type MountProxy struct {
	http.FileSystem
	Server           *Server
	TemplatePatterns []string
	Mounts           []Mount
	Fallback         http.FileSystem
}

func (self *MountProxy) Open(name string) (http.File, error) {
	name = strings.TrimPrefix(name, self.Server.RoutePrefix)
	log.Debugf("MountProxy: open(%q)", name)

	if mount := self.FindMountForEndpoint(name); mount != nil {
		//  return the file if it opened without fail OR if we aren't supposed to passthrough to the next mount
		if file, err := mount.Open(name); err == nil || !mount.Passthrough {
			return file, err
		}
	}

	log.Debugf("MountProxy: fallback trying '%s'", name)

	return self.Fallback.Open(name)
}

func (self *MountProxy) FindMountForEndpoint(endpointPath string) *Mount {
	if self.Mounts != nil {
		for i, mount := range self.Mounts {
			if mount.WillRespondTo(endpointPath) {
				log.Debugf("MountProxy: mount[%d] '%s' responding authoritatively to %s", i, mount.MountPoint, endpointPath)
				return &mount
			}
		}
	}

	return nil
}

func (self *MountProxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if tmpl, err := self.GetTemplateFromRequest(req); err == nil {
		payload := map[string]interface{}{
			`request`: req,
			`server`:  self.Server,
		}

		for _, binding := range self.Server.Config.Bindings {
			if binding.ShouldEvaluate(req) {
				if _, ok := payload[binding.Name]; !ok {
					if value, err := binding.Evaluate(req); err == nil && value != nil {
						payload[binding.Name] = value
					} else {
						log.Warningf("Binding %q failed: %v", binding.Name, err)
					}
				} else {
					log.Warningf("Binding %q failed: key already exists in payload", binding.Name)
				}
			}
		}

		log.Debugf("Rendering template %q", tmpl.Name())

		if err := tmpl.Execute(w, payload); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	} else {
		http.FileServer(self).ServeHTTP(w, req)
	}
}

func (self *MountProxy) GetTemplateFromRequest(req *http.Request) (*template.Template, error) {
	endpointPath := strings.TrimPrefix(req.URL.Path, self.Server.RoutePrefix)
	pathPrefix := ``
	mountPrefix := ``

	if strings.HasSuffix(endpointPath, `/`) {
		endpointPath = endpointPath + `index.html`
	}

	shouldExit := true

	if self.TemplatePatterns != nil {
		for _, pattern := range self.TemplatePatterns {
			if matches, err := filepath.Match(pattern, path.Base(endpointPath)); err == nil {
				if matches {
					shouldExit = false
					break
				}
			} else {
				log.Warningf("Invalid template match pattern %q: %v", pattern, err)
			}
		}
	}

	if shouldExit {
		return nil, fmt.Errorf("Request path %q is not a candidate for templating", endpointPath)
	}

	if mount := self.FindMountForEndpoint(endpointPath); mount != nil {
		pathPrefix = mount.Path
		mountPrefix = mount.MountPoint
		cwd, err := os.Getwd()

		if pathPrefix == `./` && err == nil {
			pathPrefix = cwd
		}
	} else {
		pathPrefix = self.Server.RootPath
	}

	endpointPath = path.Join(pathPrefix, strings.TrimPrefix(endpointPath, mountPrefix))

	if templatePath, err := filepath.Abs(endpointPath); err == nil {
		if !strings.HasPrefix(templatePath, pathPrefix) {
			return nil, fmt.Errorf("Template must be under directory %q (got: %q)", pathPrefix, templatePath)
		}

		templateName := path.Base(templatePath)

		return template.New(templateName).ParseFiles(templatePath)
	} else {
		return nil, err
	}
}
