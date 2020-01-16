package diecast

import (
	"net/http"
	"regexp"

	"github.com/yosssi/gohtml"
)

var rxEmptyLine = regexp.MustCompile(`(?m)^\s*$[\r\n]*|[\r\n]+\s+\z`)

type PostprocessorFunc func(string, *http.Request) (string, error)

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

func TrimEmptyLines(in string, req *http.Request) (string, error) {
	return rxEmptyLine.ReplaceAllString(in, ``) + "\n", nil
}

func PrettifyHTML(in string, req *http.Request) (string, error) {
	return gohtml.Format(in), nil
}
