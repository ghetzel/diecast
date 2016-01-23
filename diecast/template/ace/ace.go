package ace

import (
    "fmt"
    "html/template"
    "io"
    "os"
    "github.com/yosssi/ace"
    diecast "github.com/ghetzel/diecast/diecast/template"
    "github.com/ghetzel/diecast/diecast/functions"
)

type AceTemplate struct {
    diecast.Template

    template *template.Template
}


func New() diecast.ITemplate {
    return &AceTemplate{}
}

func (self *AceTemplate) Load(key string) error {
    innerTpl := fmt.Sprintf("%s/%s", self.GetTemplateDir(), key)

    if _, err := os.Stat(innerTpl + `.ace`); err == nil {
        aceKey := key

        if key == `index` {
            aceKey = ``
        }

        if tpl, err := ace.Load(`index`, aceKey, &ace.Options{
            DynamicReload: true,
            BaseDir:       self.GetTemplateDir(),
            FuncMap:       functions.GetBaseFunctions(),
        }); err == nil {
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


func (self *AceTemplate) Render(output io.Writer, payload interface{}) error {
    if self.template != nil {
        return self.template.Execute(output, payload)
    }else{
        return fmt.Errorf("Cannot execute nil template")
    }
}
