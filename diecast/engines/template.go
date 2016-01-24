package engines

import (
    "io"
)

const DEFAULT_TEMPLATE_PATH = `templates`

type ITemplate interface {
    Load(string) error
    GetTemplateDir() string
    SetTemplateDir(string)
    Render(io.Writer, map[string]interface{}) error
}


type Template struct {
    ITemplate

    TemplateDir string
}

func (self *Template) SetTemplateDir(path string) {
    self.TemplateDir = path
}

func (self *Template) GetTemplateDir() string {
    if self.TemplateDir == `` {
        self.SetTemplateDir(DEFAULT_TEMPLATE_PATH)
    }

    return self.TemplateDir
}