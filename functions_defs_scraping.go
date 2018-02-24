package diecast

import (
	"fmt"

	"github.com/PuerkitoBio/goquery"
)

func loadStandardFunctionsWebScraping(rv FuncMap) {
	// fn hquery: Queries a given HTML **document** (as returned by a Binding) and returns a list of
	//            Elements matching the given **selector**
	rv[`hquery`] = func(docI interface{}, selector string) ([]map[string]interface{}, error) {
		elements := make([]map[string]interface{}, 0)

		if doc, ok := docI.(*goquery.Document); ok {
			if doc != nil {
				doc.Find(selector).Each(func(i int, match *goquery.Selection) {
					if len(match.Nodes) > 0 {
						for _, node := range match.Nodes {
							if nodeData := htmlNodeToMap(node); len(nodeData) > 0 {
								elements = append(elements, nodeData)
							}
						}
					}
				})
			}
		} else {
			return nil, fmt.Errorf("Expected a HTML document string or object, got: %T", docI)
		}

		return elements, nil
	}
}
