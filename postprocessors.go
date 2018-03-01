package diecast

import "regexp"

var rxEmptyLine = regexp.MustCompile(`(?m)^\s*$[\r\n]*|[\r\n]+\s+\z`)

type PostprocessorFunc func(string) (string, error)

var registeredPostprocessors = map[string]PostprocessorFunc{
	`trim-empty-lines`: TrimEmptyLines,
}

func RegisterPostprocessor(name string, ppfunc PostprocessorFunc) {
	if ppfunc != nil {
		registeredPostprocessors[name] = ppfunc
	}
}

func TrimEmptyLines(in string) (string, error) {
	return rxEmptyLine.ReplaceAllString(in, ``) + "\n", nil
}
