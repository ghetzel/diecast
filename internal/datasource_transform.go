package internal

import (
	"fmt"

	"github.com/ghetzel/go-stockutil/sliceutil"
)

type DataTransforms []*DataTransform

func (self DataTransforms) Transform(data any) (any, error) {
	for _, transform := range self {
		if d, err := transform.Transform(data); err == nil {
			data = d
		} else {
			return data, err
		}
	}

	return data, nil
}

type DataTransform struct {
	Type       string `yaml:"type"`
	Expression any    `yaml:"expr"`
}

func (self *DataTransform) Transform(data any) (any, error) {
	switch self.Type {
	case ``, `jsonata`:
		return applyJsonata(data, nil, sliceutil.Sliceify(self.Expression)...)
	default:
		return data, fmt.Errorf("undefined transformation %q", self.Type)
	}
}
