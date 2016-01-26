package diecast

import (
    "github.com/ghodss/yaml"
)

type Config struct {
    Bindings map[string]BindingConfig `json:"bindings"`
}

func LoadConfig(data []byte) (Config, error) {
    rv := Config{}
    err := yaml.Unmarshal(data, &rv)
    return rv, err
}
