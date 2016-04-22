package diecast

import (
	"fmt"
	"github.com/ghetzel/diecast/diecast/engines"
	"github.com/ghetzel/diecast/diecast/engines/mustache"
	"github.com/ghetzel/diecast/diecast/engines/pongo"
	"io"
	"os"
	"path"
	"strings"
)

var DefaultTemplateEngine string = `golang`

type Route struct {
	Index        int               `json:"index"`
	Path         string            `json:"path"`
	Methods      []string          `json:"methods,omitempty"`
	Params       map[string]string `json:"params,omitempty"`
	Headers      map[string]string `json:"headers"`
	Engine       string            `json:"engine"`
	Bindings     []string          `json:"bindings"`
	TemplateName string            `json:"template"`
	Final        bool              `json:"final,omitempty"`
	basePath     string
	templateKey  string
	templatePath string
	template     engines.ITemplate
}

func (self *Route) Initialize() error {
	if self.Engine == `` {
		self.Engine = DefaultTemplateEngine
	}

	if self.Methods == nil || len(self.Methods) == 0 {
		self.Methods = []string{`GET`}
	}

	if self.Path == `/` {
		self.TemplateName = `index.html`
	}

	return nil
}

func (self *Route) LoadTemplate(basePath string) error {
	if err := pongo.Initialize(); err != nil {
		return err
	}

	if err := mustache.Initialize(); err != nil {
		return err
	}

	self.basePath = basePath

	//  figure out the template path
	if self.TemplateName == `` {
		self.templatePath = path.Join(self.basePath, fmt.Sprintf("%s.%s", self.Path, strings.ToLower(self.Engine)))
	} else if strings.HasPrefix(self.TemplateName, `/`) {
		self.templatePath = self.TemplateName
	} else {
		self.templatePath = path.Join(self.basePath, fmt.Sprintf("%s.%s", self.TemplateName, strings.ToLower(self.Engine)))
	}

	//  figure out what this template's unique key will be
	self.templateKey = strings.TrimSuffix(strings.TrimPrefix(self.templatePath, path.Clean(self.basePath)+`/`), path.Ext(self.templatePath))

	//  get and instance of this template engine
	switch self.Engine {
	case `mustache`:
		self.template = mustache.New()
	case `pongo`:
		self.template = pongo.New()
	default:
		return fmt.Errorf("Unknown template engine '%s'", self.Engine)
	}

	//  attempt to stat the template file
	if _, err := os.Stat(self.templatePath); err == nil {
		self.template.SetTemplateDir(self.basePath)
	} else {
		return err
	}

	if err := self.template.Load(self.templateKey); err != nil {
		return fmt.Errorf("Error loading template '%s': %v", self.templatePath, err)
	}

	return nil
}

func (self *Route) Render(writer io.Writer, payload map[string]interface{}) error {
	return self.template.Render(writer, payload)
}
