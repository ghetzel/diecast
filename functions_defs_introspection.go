package diecast

import (
	"fmt"
	"os"
	"strings"

	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/stringutil"
)

func loadStandardFunctionsIntrospection(_ FuncMap, _ *Server) funcGroup {
	return funcGroup{
		Name:        `Introspection and Reflection`,
		Description: `Functions for inspecting runtime information about templates and Diecast itself.`,
		Functions: []funcDef{
			{
				Name:    `templateKey`,
				Summary: `Open the given file and retrieve the key from the page object defined in its header.`,
				Function: func(filenameI any, keyI any, fallbacks ...any) (any, error) {
					if filename, err := stringutil.ToString(filenameI); err == nil {
						if key, err := stringutil.ToString(keyI); err == nil {
							if file, err := os.Open(filename); err == nil {
								var fallback any

								if values := sliceutil.Sliceify(sliceutil.Stringify(fallbacks)); len(values) > 0 {
									fallback = values[0]
								}

								if header, _, err := SplitTemplateHeaderContent(file); err == nil && header != nil {
									return maputil.DeepGet(header.Page, strings.Split(key, `.`), fallback), nil
								}

								return fallback, nil
							} else {
								return nil, err
							}
						} else {
							return nil, fmt.Errorf("unable to convert key to string")
						}
					} else {
						return nil, fmt.Errorf("unable to convert filename to string")
					}
				},
			},
		},
	}
}
