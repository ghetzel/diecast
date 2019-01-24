package diecast

import (
	"encoding/hex"

	"github.com/ghetzel/go-stockutil/colorutil"
	"github.com/ghetzel/go-stockutil/typeutil"
	"github.com/spaolacci/murmur3"
)

func loadStandardFunctionsColor(funcs FuncMap) funcGroup {
	return funcGroup{
		Name:        `Color Manipulation`,
		Description: `Used to parse, manipulate, and adjust string representations of visible colors.`,
		Functions: []funcDef{
			{
				Name:    `lighten`,
				Summary: `Lighten the given color by a percent [0-100].`,
				Function: func(color interface{}, percent float64) (string, error) {
					if c, err := colorutil.Lighten(color, int(percent)); err == nil {
						return c.String(), nil
					} else {
						return ``, err
					}
				},
			}, {
				Name:    `darken`,
				Summary: `Darken the given color by a percent [0-100].`,
				Function: func(color interface{}, percent float64) (string, error) {
					if c, err := colorutil.Darken(color, int(percent)); err == nil {
						return c.String(), nil
					} else {
						return ``, err
					}
				},
			}, {
				Name:    `colorToHex`,
				Summary: `Convert the given color to a hexadecimal ("#RRGGBB", "#RRGGBBAA") value.`,
				Function: func(color interface{}) (string, error) {
					if c, err := colorutil.Parse(color); err == nil {
						return c.String(), nil
					} else {
						return ``, err
					}
				},
			}, {
				Name:    `colorToRGB`,
				Summary: `Convert the given color to an "rgb()" or "rgba()" value.`,
				Function: func(color interface{}) (string, error) {
					if c, err := colorutil.Parse(color); err == nil {
						return c.StringRGBA(), nil
					} else {
						return ``, err
					}
				},
			}, {
				Name:    `colorToHSL`,
				Summary: `Convert the given color to an "hsl()" or "hsla()" value.`,
				Function: func(color interface{}) (string, error) {
					if c, err := colorutil.Parse(color); err == nil {
						return c.StringHSLA(), nil
					} else {
						return ``, err
					}
				},
			}, {
				Name:    `colorFromValue`,
				Summary: `Consistently generate the same color from color from a given value.`,
				Function: func(value interface{}) string {
					mmh3 := murmur3.New64().Sum([]byte(typeutil.V(value).String()))

					if len(mmh3) >= 3 {
						return `#` + hex.EncodeToString(mmh3[0:3])
					} else {
						return `#000000`
					}
				},
			},
		},
	}
}
