package diecast

import (
	"encoding/hex"
	"strings"

	"github.com/ghetzel/go-stockutil/colorutil"
	"github.com/ghetzel/go-stockutil/typeutil"
	"github.com/spaolacci/murmur3"
)

func loadStandardFunctionsColor(funcs FuncMap, server *Server) funcGroup {
	return funcGroup{
		Name:        `Color Manipulation`,
		Description: `Used to parse, manipulate, and adjust string representations of visible colors.`,
		Functions: []funcDef{
			{
				Name:    `lighten`,
				Summary: `Lighten the given color by a percent [0-100].`,
				Arguments: []funcArg{
					{
						Name:        `color`,
						Type:        `string`,
						Description: `The base color to operate on.  Can be specified as rgb(), rgba(), hsl(), hsla(), hsv(), hsva(), "#RRGGBB", or "#RRGGBBAA".`,
					}, {
						Name:        `percent`,
						Type:        `number`,
						Description: `A how much lighter the color should be made.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `lighten "#FF0000" 15`,
						Return: `#FF4D4D`,
					},
				},
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
				Arguments: []funcArg{
					{
						Name:        `color`,
						Type:        `string`,
						Description: `The base color to operate on.  Can be specified as rgb(), rgba(), hsl(), hsla(), hsv(), hsva(), "#RRGGBB", or "#RRGGBBAA".`,
					}, {
						Name:        `percent`,
						Type:        `number`,
						Description: `A how much darker the color should be made.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `darken "#FF0000" 15`,
						Return: `#B30000`,
					},
				},
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
				Arguments: []funcArg{
					{
						Name:        `color`,
						Type:        `string`,
						Description: `The color to convert.  Can be specified as rgb(), rgba(), hsl(), hsla(), hsv(), hsva(), "#RRGGBB", or "#RRGGBBAA".`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `colorToHex "hsl(0, 100%, 50%)"`,
						Return: `#FF0000`,
					},
				},
				Function: func(color interface{}) (string, error) {
					if c, err := colorutil.Parse(color); err == nil {
						return c.String(), nil
					} else {
						return ``, err
					}
				},
			}, {
				Name: `colorToRGB`,
				Arguments: []funcArg{
					{
						Name:        `color`,
						Type:        `string`,
						Description: `The color to convert.  Can be specified as rgb(), rgba(), hsl(), hsla(), hsv(), hsva(), "#RRGGBB", or "#RRGGBBAA".`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `colorToHex "#FF0000"`,
						Return: `rgb(255, 0, 0)`,
					},
				},
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
				Arguments: []funcArg{
					{
						Name:        `color`,
						Type:        `string`,
						Description: `The color to convert.  Can be specified as rgb(), rgba(), hsl(), hsla(), hsv(), hsva(), "#RRGGBB", or "#RRGGBBAA".`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `colorToHex "#FF0000"`,
						Return: `hsl(0, 100%, 50%)`,
					},
				},
				Function: func(color interface{}) (string, error) {
					if c, err := colorutil.Parse(color); err == nil {
						return c.StringHSLA(), nil
					} else {
						return ``, err
					}
				},
			}, {
				Name: `colorFromValue`,
				Summary: `Consistently generate the same color from a given value of any type. This can be useful for ` +
					`providing automatically generated colors to represent data for which a set of predefined colors ` +
					`may not have significant meaning (for example: user avatars and contact lists).`,
				Arguments: []funcArg{
					{
						Name:        `value`,
						Type:        `any`,
						Description: `A value to generate a color for.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `colorFromValue "Alice"`,
						Return: `#416C69`,
					},
				},
				Function: func(value interface{}) string {
					mmh3 := murmur3.New64().Sum([]byte(typeutil.V(value).String()))

					if len(mmh3) >= 3 {
						return `#` + strings.ToUpper(hex.EncodeToString(mmh3[0:3]))
					} else {
						return `#000000`
					}
				},
			},
		},
	}
}
