package diecast

import (
	"encoding/json"
	"fmt"
	"html/template"

	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday/v2"
)

func loadStandardFunctionsCodecs(rv FuncMap) {
	// fn jsonify: Encode the given *value* as a JSON string, optionally using *indent* to pretty
	//             format the output.
	rv[`jsonify`] = func(value interface{}, indent ...string) (string, error) {
		indentString := ``

		if len(indent) > 0 {
			indentString = indent[0]
		}

		data, err := json.MarshalIndent(value, ``, indentString)
		return string(data[:]), err
	}

	// fn markdown: Render the given Markdown string *value* as sanitized HTML.
	rv[`markdown`] = func(value interface{}, extensions ...string) (template.HTML, error) {
		input := fmt.Sprintf("%v", value)
		output := blackfriday.Run(
			[]byte(input),
			blackfriday.WithExtensions(toMarkdownExt(extensions...)),
		)
		output = bluemonday.UGCPolicy().SanitizeBytes(output)

		return template.HTML(output[:]), nil
	}

	// fn csv: Render the given *values* as a line suitable for inclusion in a common-separated
	//         values file.
	rv[`csv`] = func(header []interface{}, lines []interface{}) (string, error) {
		return delimited(',', header, lines)
	}

	// fn tsv: Render the given *values* as a line suitable for inclusion in a tab-separated
	//         values file.
	rv[`tsv`] = func(header []interface{}, lines []interface{}) (string, error) {
		return delimited('\t', header, lines)
	}

	// fn unsafe: Return an unescaped raw HTML segment for direct inclusion in the rendered
	//            template output.  This is a common antipattern that leads to all kinds of
	//            security issues from poorly-constrained implementations, so you are forced
	//            to acknowledge this by typing "unsafe".
	rv[`unsafe`] = func(value string) template.HTML {
		return template.HTML(value)
	}

	// fn sanitize: Takes a raw HTML string and santizes it, removing attributes and elements
	//              that can be used to evaluate scripts, but leaving the rest.  Useful for
	//              preparing user-generated HTML for display.
	rv[`sanitize`] = func(value string) template.HTML {
		return template.HTML(bluemonday.UGCPolicy().Sanitize(value))
	}
}
