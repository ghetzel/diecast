package diecast

import (
	"fmt"
	"os"
	"strings"

	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/stringutil"
)

func loadStandardFunctionsIntrospection(rv FuncMap) {
	// fn templateKey: Open the given *file* and retrieve the *key* from the page object in its header.
	rv[`templateKey`] = func(filenameI interface{}, keyI interface{}, fallbacks ...interface{}) (interface{}, error) {
		if filename, err := stringutil.ToString(filenameI); err == nil {
			if key, err := stringutil.ToString(keyI); err == nil {
				if file, err := os.Open(filename); err == nil {
					var fallback interface{}

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
				return nil, fmt.Errorf("Unable to convert key to string")
			}
		} else {
			return nil, fmt.Errorf("Unable to convert filename to string")
		}
	}

}
