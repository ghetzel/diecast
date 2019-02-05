package diecast

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	strip "github.com/grokify/html-strip-tags-go"
	htmlmain "html"
	"html/template"
)

func loadStandardFunctionsHtmlProcessing(funcs FuncMap) funcGroup {
	return funcGroup{
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
				Name:    `htmlquery`,
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
					},
				},
				Function: func(docI interface{}, selector string, name string, value interface{}) (template.HTML, error) {
					return htmlModify(docI, selector, `set-attr`, name, value)
				},
			},
		},
	}
}
