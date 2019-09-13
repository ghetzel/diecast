package diecast

import (
	"regexp"

	"github.com/yosssi/gohtml"
)

var rxEmptyLine = regexp.MustCompile(`(?m)^\s*$[\r\n]*|[\r\n]+\s+\z`)

type PostprocessorFunc func(string) (string, error)

func init() {
	RegisterPostprocessor(`trim-empty-lines`, TrimEmptyLines)
	RegisterPostprocessor(`prettify-html`, PrettifyHTML)
}

var registeredPostprocessors = make(map[string]PostprocessorFunc)

func RegisterPostprocessor(name string, ppfunc PostprocessorFunc) {
	if ppfunc != nil {
		registeredPostprocessors[name] = ppfunc
	}
}

func TrimEmptyLines(in string) (string, error) {
	return rxEmptyLine.ReplaceAllString(in, ``) + "\n", nil
}

func PrettifyHTML(in string) (string, error) {
	return gohtml.Format(in), nil
}
