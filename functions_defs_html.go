package diecast

import (
	"fmt"
	htmlmain "html"
	"html/template"

	"github.com/PuerkitoBio/goquery"
	strip "github.com/grokify/html-strip-tags-go"
)

func loadStandardFunctionsHtmlProcessing(funcs FuncMap, server *Server) funcGroup {
	group := funcGroup{
		Name:        `HTML Processing`,
		Description: `Used to parse and modify HTML documents.`,
		Functions: []funcDef{
			{
				Name: `stripHtml`,
				Summary: `Removes all HTML tags from a given input string, leaving behind only the ` +
					`textual content of the nodes. Only text nodes are preserved; attribute names ` +
					`and values, and comments, will be omitted.`,
				Function: func(in interface{}) string {
					stripped := strip.StripTags(fmt.Sprintf("%v", in))
					stripped = htmlmain.UnescapeString(stripped)
					return stripped
				},
			}, {
				Name: `htmlQuery`,
				Aliases: []string{
					`htmlquery`,
				},
				Summary: `Parse a given HTML document and return details about all elements matching a CSS selector.`,
				Arguments: []funcArg{
					{
						Name:        `document`,
						Type:        `string`,
						Description: `The HTML document to parse.`,
					}, {
						Name:        `selector`,
						Type:        `string`,
						Description: `A CSS selector that targets the elements that will be returned.`,
					},
				},
				Function: func(docI interface{}, selector string) ([]map[string]interface{}, error) {
					elements := make([]map[string]interface{}, 0)

					if doc, err := htmldoc(docI); err == nil {
						doc.Find(selector).Each(func(i int, match *goquery.Selection) {
							if len(match.Nodes) > 0 {
								for _, node := range match.Nodes {
									if nodeData := htmlNodeToMap(node); len(nodeData) > 0 {
										elements = append(elements, nodeData)
									}
								}
							}
						})
					} else {
						return nil, err
					}

					return elements, nil
				},
			}, {
				Name:    `htmlRemove`,
				Summary: `Parse a given HTML document and remove all elements matching a CSS selector.`,
				Arguments: []funcArg{
					{
						Name:        `document`,
						Type:        `string`,
						Description: `The HTML document to parse.`,
					}, {
						Name:        `selector`,
						Type:        `string`,
						Description: `A CSS selector that targets the elements that will be returned.`,
					},
				},
				Function: func(docI interface{}, selector string) (template.HTML, error) {
					return htmlModify(docI, selector, `remove`, ``, nil)
				},
			}, {
				Name:    `htmlAddClass`,
				Summary: `Parse a given HTML document and add a CSS class to all elements matching a CSS selector.`,
				Arguments: []funcArg{
					{
						Name:        `document`,
						Type:        `string`,
						Description: `The HTML document to parse.`,
					}, {
						Name:        `selector`,
						Type:        `string`,
						Description: `A CSS selector that targets the elements that will be returned.`,
					},
				},
				Function: func(docI interface{}, selector string, classes ...string) (template.HTML, error) {
					return htmlModify(docI, selector, `add-class`, ``, classes)
				},
			}, {
				Name:    `htmlRemoveClass`,
				Summary: `Parse a given HTML document and remove a CSS class to all elements matching a CSS selector.`,
				Arguments: []funcArg{
					{
						Name:        `document`,
						Type:        `string`,
						Description: `The HTML document to parse.`,
					}, {
						Name:        `selector`,
						Type:        `string`,
						Description: `A CSS selector that targets the elements that will be returned.`,
					},
				},
				Function: func(docI interface{}, selector string, classes ...string) (template.HTML, error) {
					return htmlModify(docI, selector, `remove-class`, ``, classes)
				},
			}, {
				Name:    `htmlSetAttr`,
				Summary: `Parse a given HTML document and set an attribute to a given value on all elements matching a CSS selector.`,
				Arguments: []funcArg{
					{
						Name:        `document`,
						Type:        `string`,
						Description: `The HTML document to parse.`,
					}, {
						Name:        `selector`,
						Type:        `string`,
						Description: `A CSS selector that targets the elements that will be returned.`,
					}, {
						Name:        `attribute`,
						Type:        `string`,
						Description: `The name of the attribute to modify on matching elements.`,
					}, {
						Name:        `value`,
						Type:        `any`,
						Description: `The value to set the matching attributes to.`,
					},
				},
				Function: func(docI interface{}, selector string, name string, value interface{}) (template.HTML, error) {
					return htmlModify(docI, selector, `set-attr`, name, value)
				},
			}, {
				Name: `htmlAttrFindReplace`,
				Summary: `Parse a given HTML document and locate a set of elements. For the given attribute name, ` +
					`perform a find and replace operation on the values.`,
				Arguments: []funcArg{
					{
						Name:        `document`,
						Type:        `string`,
						Description: `The HTML document to parse.`,
					}, {
						Name:        `selector`,
						Type:        `string`,
						Description: `A CSS selector that targets the elements that will be modified.`,
					}, {
						Name:        `attribute`,
						Type:        `string`,
						Description: `The name of the attribute to modify on matching elements.`,
					}, {
						Name:        `find`,
						Type:        `string`,
						Description: `A regular expression that will be used to find matching text in the affected attributes.`,
					}, {
						Name: `replace`,
						Type: `string`,
						Description: `The value that will replace any found text.  Capture groups in the regular ` +
							`expression can be referenced using a "$", e.g.: ${1}, ${2}, ${name}.`,
					},
				},
				Function: func(document interface{}, selector string, attribute string, find string, replace interface{}) (template.HTML, error) {
					return htmlModify(document, selector, `find-replace-attr`, attribute, replace, find)
				},
			}, {
				Name: `htmlTextFindReplace`,
				Summary: `Parse a given HTML document and locate a set of elements. For each matched element, ` +
					`perform a find and replace operation on the text content of the element (including all descendants).`,
				Arguments: []funcArg{
					{
						Name:        `document`,
						Type:        `string`,
						Description: `The HTML document to parse.`,
					}, {
						Name:        `selector`,
						Type:        `string`,
						Description: `A CSS selector that targets the elements that will be modified.`,
					}, {
						Name:        `find`,
						Type:        `string`,
						Description: `A regular expression that will be used to find matching text in the affected attributes.`,
					}, {
						Name: `replace`,
						Type: `string`,
						Description: `The value that will replace any found text.  Capture groups in the regular ` +
							`expression can be referenced using a "$", e.g.: ${1}, ${2}, ${name}.`,
					},
				},
				Function: func(document interface{}, selector string, find string, replace interface{}) (template.HTML, error) {
					return htmlModify(document, selector, `find-replace-text`, ``, replace, find)
				},
			}, {
				Name:    `htmlInner`,
				Summary: `Parse a given HTML document and return the HTML content of the first element matching the given CSS selector.`,
				Arguments: []funcArg{
					{
						Name:        `document`,
						Type:        `string`,
						Description: `The HTML document to parse.`,
					}, {
						Name:        `selector`,
						Type:        `string`,
						Description: `A CSS selector that targets the element whose contents will be returned.`,
					},
				},
				Function: func(docI interface{}, selector string) (template.HTML, error) {
					if doc, err := htmldoc(docI); err == nil {
						if contents, err := doc.Find(selector).Html(); err == nil {
							return template.HTML(contents), nil
						} else {
							return ``, err
						}
					} else {
						return ``, err
					}
				},
			},
		},
	}

	group.Functions = append(group.Functions, []funcDef{
		{
			Name:     `htmlquery`,
			Alias:    `htmlQuery`,
			Function: group.fn(`htmlQuery`),
			Hidden:   true,
		},
	}...)

	return group
}
