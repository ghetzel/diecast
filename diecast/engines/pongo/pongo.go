package pongo

import (
    "fmt"
    "io"
    "os"
    "github.com/flosch/pongo2"
    "github.com/ghetzel/diecast/diecast/engines"
    // "github.com/ghetzel/diecast/diecast/functions"
)

type PongoTemplate struct {
    engines.Template

    template *pongo2.Template
}


func New() engines.ITemplate {
    return &PongoTemplate{}
}

func (self *PongoTemplate) Load(key string) error {
    tplPath := fmt.Sprintf("%s/%s.pongo", self.GetTemplateDir(), key)

    if _, err := os.Stat(tplPath); err == nil {
        if tpl, err := pongo2.FromFile(tplPath); err == nil {
            self.template = tpl
            return nil
        }else{
            return err
        }
        return nil
    }else{
        return err
    }
}


func (self *PongoTemplate) Render(output io.Writer, payload map[string]interface{}) error {
    if self.template != nil {
        return self.template.ExecuteWriter(pongo2.Context(payload), output)
    }else{
        return fmt.Errorf("Cannot execute nil template")
    }
}
