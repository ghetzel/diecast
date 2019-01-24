package diecast

import (
	"github.com/PuerkitoBio/goquery"
	"html/template"
)

func loadStandardFunctionsWebScraping(rv FuncMap) {
	rv[`htmlquery`] = func(docI interface{}, selector string) ([]map[string]interface{}, error) {
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
	}

	rv[`htmlRemove`] = func(docI interface{}, selector string) (template.HTML, error) {
		return htmlModify(docI, selector, `remove`, ``, nil)
	}

	rv[`htmlAddClass`] = func(docI interface{}, selector string, classes ...string) (template.HTML, error) {
		return htmlModify(docI, selector, `add-class`, ``, classes)
	}

	rv[`htmlRemoveClass`] = func(docI interface{}, selector string, classes ...string) (template.HTML, error) {
		return htmlModify(docI, selector, `remove-class`, ``, classes)
	}

	rv[`htmlSetAttr`] = func(docI interface{}, selector string, name string, value interface{}) (template.HTML, error) {
		return htmlModify(docI, selector, `set-attr`, name, value)
	}
}
