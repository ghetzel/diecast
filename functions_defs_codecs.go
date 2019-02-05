package diecast

import (
	"encoding/json"
	"fmt"
	"html/template"

	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday/v2"
)

func loadStandardFunctionsCodecs(funcs FuncMap) funcGroup {
	return funcGroup{
		Name:        `Encoding and Decoding`,
		Description: `For encoding typed data and data structures into well-known formats like JSON, CSV, and TSV.`,
		Functions: []funcDef{
			{
				Name:    `jsonify`,
				Summary: `Encode the given argument as a JSON document.`,
				Arguments: []funcArg{
					{
						Name:        `data`,
						Type:        `any`,
						Description: `The data to encode as JSON.`,
					}, {
						Name:        `indent`,
						Type:        `string`,
						Optional:    true,
						Default:     `  `,
						Description: `The string to indent successive tiers in the document hierarchy with.`,
					},
				},
				Function: func(value interface{}, indent ...string) (string, error) {
					indentString := `  `

					if len(indent) > 0 {
						indentString = indent[0]
					}

					data, err := json.MarshalIndent(value, ``, indentString)
					return string(data[:]), err
				},
			}, {
				// fn markdown: Render the given Markdown string *value* as sanitized HTML.
				Name:    `markdown`,
				Summary: `Parse the given string as a Markdown document and return it represented as HTML.`,
				Arguments: []funcArg{
					{
						Name:        `document`,
						Type:        `string`,
						Description: `The full text of the Markdown to parse`,
					}, {
						Name:        `extensions`,
						Type:        `string(s)`,
						Description: `A list of zero of more Markdown extensions to enable when rendering the HTML.`,
						Valid: []funcArg{
							{
								Name:        `no-intra-emphasis`,
								Description: ``,
							}, {
								Name:        `tables`,
								Description: ``,
							}, {
								Name:        `fenced-code`,
								Description: ``,
							}, {
								Name:        `autolink`,
								Description: ``,
							}, {
								Name:        `strikethrough`,
								Description: ``,
							}, {
								Name:        `lax-html-blocks`,
								Description: ``,
							}, {
								Name:        `space-headings`,
								Description: ``,
							}, {
								Name:        `hard-line-break`,
								Description: ``,
							}, {
								Name:        `tab-size-eight`,
								Description: ``,
							}, {
								Name:        `footnotes`,
								Description: ``,
							}, {
								Name:        `no-empty-line-before-block`,
								Description: ``,
							}, {
								Name:        `heading-ids`,
								Description: ``,
							}, {
								Name:        `titleblock`,
								Description: ``,
							}, {
								Name:        `auto-heading-ids`,
								Description: ``,
							}, {
								Name:        `backslash-line-break`,
								Description: ``,
							}, {
								Name:        `definition-lists`,
								Description: ``,
							}, {
								Name:        `common`,
								Description: ``,
							},
						},
						Variadic: true,
					},
				},
				Function: func(value interface{}, extensions ...string) (template.HTML, error) {
					input := fmt.Sprintf("%v", value)
					output := blackfriday.Run(
						[]byte(input),
						blackfriday.WithExtensions(toMarkdownExt(extensions...)),
					)
					output = bluemonday.UGCPolicy().SanitizeBytes(output)

					return template.HTML(output[:]), nil
				},
			}, {
				Name:    `csv`,
				Summary: `Encode the given data as a comma-separated values document.`,
				Arguments: []funcArg{
					{
						Name:        `columns`,
						Type:        `array[string]`,
						Description: `An array of values that represent the column names of the table being created.`,
					}, {
						Name:        `rows`,
						Type:        `array[array[string]], array[object]`,
						Description: `An array of values that represent the column names of the table being created.`,
					},
				},
				Function: func(columns []interface{}, rows []interface{}) (string, error) {
					return delimited(',', columns, rows)
				},
			}, {
				Name:    `tsv`,
				Summary: `Encode the given data as a tab-separated values document.`,
				Arguments: []funcArg{
					{
						Name:        `columns`,
						Type:        `array[string]`,
						Description: `An array of values that represent the column names of the table being created.`,
					}, {
						Name:        `rows`,
						Type:        `array[array[string]], array[object]`,
						Description: `An array of values that represent the column names of the table being created.`,
					},
				},
				Function: func(columns []interface{}, rows []interface{}) (string, error) {
					return delimited('\t', columns, rows)
				},
			}, {
				Name: `unsafe`,
				Summary: `Return an unescaped raw HTML segment for direct inclusion in the rendered template output.` +
					`This function bypasses the built-in HTML escaping and sanitization security features, and you ` +
					`almost certainly want to use [sanitize](#fn-sanitize) instead.  This is a common antipattern that ` +
					`leads to all kinds of security issues from poorly-constrained implementations, so you are forced ` +
					`to acknowledge this by typing "unsafe".`,
				Arguments: []funcArg{
					{
						Name:        `document`,
						Type:        `string`,
						Description: `The raw HTML snippet you sneakily want to sneak past the HTML sanitizer for reasons.`,
					},
				},
				Function: func(value string) template.HTML {
					return template.HTML(value)
				},
			}, {
				Name: `sanitize`,
				Summary: `Takes a raw HTML string and santizes it, removing attributes and elements that can be used ` +
					`to evaluate scripts, but leaving the rest. Useful for preparing user-generated HTML for display.`,
				Arguments: []funcArg{
					{
						Name:        ``,
						Type:        ``,
						Description: ``,
					},
				},
				Function: func(value string) template.HTML {
					return template.HTML(bluemonday.UGCPolicy().Sanitize(value))
				},
			},
		},
	}
}
