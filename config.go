package diecast

import (
	"github.com/ghodss/yaml"
)

type Config struct {
	TemplatePatterns []string  `json:"template_patterns"`
	Bindings         []Binding `json:"bindings,omitempty"`
	Mounts           []Mount   `json:"mounts,omitempty"`
}

type GlobalConfig struct {
	DefaultEngine string                 `json:"default_engine,omitempty"`
	Headers       map[string]string      `json:"headers,omitempty"`
	Payload       map[string]interface{} `json:"payload,omitempty"`
}

func LoadConfig(data []byte) (Config, error) {
	rv := Config{}
	err := yaml.Unmarshal(data, &rv)
	return rv, err
}
