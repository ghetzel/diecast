package diecast

import (
    "github.com/ghodss/yaml"
)

type BindingConfig struct {
    Routes         []string          `json:"routes"`
    Resource       string            `json:"resource"`
    ResourceParams map[string]string `json:"params,omitempty"`
    RouteMethods   []string          `json:"route_methods,omitempty"`
    ResourceMethod string            `json:"resource_method,omitempty"`
}

type Config struct {
    Bindings map[string]BindingConfig `json:"bindings"`
}

func LoadConfig(data []byte) (Config, error) {
    rv := Config{}
    err := yaml.Unmarshal(data, &rv)
    return rv, err
}
