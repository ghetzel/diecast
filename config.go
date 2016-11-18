package diecast

import (
	"github.com/ghodss/yaml"
)

type Config struct {
	Options  GlobalConfig       `json:"options"`
	Routes   []*Route           `json:"routes"`
	Bindings map[string]Binding `json:"bindings"`
	Mounts   []Mount            `json:"mounts"`
}

type GlobalConfig struct {
	DefaultEngine string                 `json:"default_engine"`
	Headers       map[string]string      `json:"headers"`
	Payload       map[string]interface{} `json:"payload"`
}

func LoadConfig(data []byte) (Config, error) {
	rv := Config{}
	err := yaml.Unmarshal(data, &rv)
	return rv, err
}
