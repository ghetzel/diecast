package diecast

import (
	"regexp"

	"github.com/yosssi/gohtml"
)

var rxEmptyLine = regexp.MustCompile(`(?m)^\s*$[\r\n]*|[\r\n]+\s+\z`)

type PostprocessorFunc func(string) (string, error)

var registeredPostprocessors = map[string]PostprocessorFunc{
	`trim-empty-lines`: TrimEmptyLines,
	`prettify-html`:    PrettifyHTML,
}

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
