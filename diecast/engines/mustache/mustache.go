package mustache

import (
	"fmt"
	"github.com/ghetzel/diecast/diecast/engines"
	"github.com/hoisie/mustache"
	"io"
	"os"
)

type MustacheTemplate struct {
	engines.Template

	templateFilename string
	template         *mustache.Template
}

func Initialize() error {
	return nil
}

func New() engines.ITemplate {
	return &MustacheTemplate{}
}

func (self *MustacheTemplate) Load(key string) error {
	self.templateFilename = fmt.Sprintf("%s/%s.mustache", self.GetTemplateDir(), key)

	if _, err := os.Stat(self.templateFilename); err == nil {
		if tpl, err := mustache.ParseFile(self.templateFilename); err == nil {
			self.template = tpl
		} else {
			return err
		}
	} else {
		return err
	}

	return nil
}

func (self *MustacheTemplate) Render(output io.Writer, payload map[string]interface{}) error {
	if self.template != nil {
		if _, err := io.WriteString(output, self.template.Render(payload)); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("Cannot execute nil template")
	}

	return nil
}
