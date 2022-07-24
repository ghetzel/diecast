package internal

import (
	"bytes"
	"html/template"

	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/quick"
	"github.com/ghetzel/go-stockutil/typeutil"
)

var HighlightTheme = `monokai`

func loadStandardFunctionsLangHighlighting(funcs FuncMap, server ServerProxy) FuncGroup {
	var group = FuncGroup{
		Name:        `Language Highlighting`,
		Description: `Utilities for performing syntax highlighting of source code.`,
		Functions: []FuncDef{
			{
				Name:    `highlight`,
				Summary: `Take source code in the given language and return marked up HTML that highlights language keywords and syntax.`,
				Arguments: []FuncArg{
					{
						Name:        `language`,
						Type:        `string`,
						Description: `The name of the language expected as input.  If empty or the string "*", a best effort will be made to detect the language.`,
					}, {
						Name:        `src`,
						Type:        `string`,
						Description: `The source code to highlight.`,
					},
				},
				Function: func(language string, in interface{}) (template.HTML, error) {
					var out bytes.Buffer
					var src = typeutil.String(in)

					if language == `` || language == `*` {
						if c := lexers.Analyse(src).Config(); c != nil {
							language = c.Name
						}
					}

					if err := quick.Highlight(&out, typeutil.String(in), language, `html`, HighlightTheme); err == nil {
						return template.HTML(out.String()), nil
					} else {
						return ``, err
					}
				},
			},
		},
	}

	return group
}
