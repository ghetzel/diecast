package diecast

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ghetzel/go-stockutil/colorutil"
	"github.com/ghetzel/go-stockutil/typeutil"
	"github.com/spaolacci/murmur3"
)

const DefaultColorPalette = `munin`

var palettes = map[string][]string{
	`spectrum14`: {
		`#ECB796`,
		`#DC8F70`,
		`#B2A470`,
		`#92875A`,
		`#716C49`,
		`#D2ED82`,
		`#BBE468`,
		`#A1D05D`,
		`#E7CBE6`,
		`#D8AAD6`,
		`#A888C2`,
		`#9DC2D3`,
		`#649EB9`,
		`#387AA3`,
	},
	`colorwheel`: {
		`#CB513A`,
		`#73C03A`,
		`#65B9AC`,
		`#4682B4`,
		`#96557E`,
		`#785F43`,
		`#858772`,
		`#B5B6A9`,
	},
	`spectrum2000`: {
		`#57306F`,
		`#514C76`,
		`#646583`,
		`#738394`,
		`#6B9C7D`,
		`#84B665`,
		`#A7CA50`,
		`#BFE746`,
		`#E2F528`,
		`#FFF726`,
		`#ECDD00`,
		`#D4B11D`,
		`#DE8800`,
		`#DE4800`,
		`#C91515`,
		`#9A0000`,
		`#7B0429`,
		`#580839`,
		`#31082B`,
	},
	`spectrum2001`: {
		`#2F243F`,
		`#3C2C55`,
		`#4A3768`,
		`#565270`,
		`#6B6B7C`,
		`#72957F`,
		`#86AD6E`,
		`#A1BC5E`,
		`#B8D954`,
		`#D3E04E`,
		`#CCAD2A`,
		`#CC8412`,
		`#C1521D`,
		`#AD3821`,
		`#8A1010`,
		`#681717`,
		`#531E1E`,
		`#3D1818`,
		`#320A1B`,
	},
	`classic9`: {
		`#2F254A`,
		`#491D37`,
		`#7C2626`,
		`#963B20`,
		`#7D5836`,
		`#C5A32F`,
		`#DDCB53`,
		`#A2B73C`,
		`#848F39`,
		`#4A6860`,
		`#423D4F`,
	},
	`cool`: {
		`#5E9D2F`,
		`#73C03A`,
		`#4682B4`,
		`#7BC3B8`,
		`#A9884E`,
		`#C1B266`,
		`#A47493`,
		`#C09FB5`,
	},
	`munin`: {
		`#00CC00`,
		`#0066B3`,
		`#FF8000`,
		`#FFCC00`,
		`#330099`,
		`#990099`,
		`#CCFF00`,
		`#FF0000`,
		`#808080`,
		`#008F00`,
		`#00487D`,
		`#B35A00`,
		`#B38F00`,
		`#6B006B`,
		`#8FB300`,
		`#B30000`,
		`#BEBEBE`,
		`#80FF80`,
		`#80C9FF`,
		`#FFC080`,
		`#FFE680`,
		`#AA80FF`,
		`#EE00CC`,
		`#FF8080`,
		`#666600`,
		`#FFBFFF`,
		`#00FFCC`,
		`#CC6699`,
		`#999900`,
	},
}

func loadStandardFunctionsColor(_ FuncMap, _ *Server) funcGroup {
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
				Function: func(color any, percent float64) (string, error) {
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
				Function: func(color any, percent float64) (string, error) {
					if c, err := colorutil.Darken(color, int(percent)); err == nil {
						return c.String(), nil
					} else {
						return ``, err
					}
				},
			}, {
				Name:    `saturate`,
				Summary: `Saturate the given color by a percent [0-100].`,
				Arguments: []funcArg{
					{
						Name:        `color`,
						Type:        `string`,
						Description: `The base color to operate on.  Can be specified as rgb(), rgba(), hsl(), hsla(), hsv(), hsva(), "#RRGGBB", or "#RRGGBBAA".`,
					}, {
						Name:        `percent`,
						Type:        `number`,
						Description: `A how much more saturation the color should have.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `saturate "#ad4038" 20`,
						Return: `#c42b21`,
					},
				},
				Function: func(color any, percent float64) (string, error) {
					if c, err := colorutil.Saturate(color, int(percent)); err == nil {
						return c.String(), nil
					} else {
						return ``, err
					}
				},
			}, {
				Name:    `desaturate`,
				Summary: `Desaturate the given color by a percent [0-100].`,
				Arguments: []funcArg{
					{
						Name:        `color`,
						Type:        `string`,
						Description: `The base color to operate on.  Can be specified as rgb(), rgba(), hsl(), hsla(), hsv(), hsva(), "#RRGGBB", or "#RRGGBBAA".`,
					}, {
						Name:        `percent`,
						Type:        `number`,
						Description: `A how much less saturation the color should have.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `desaturate "#ad4038" 20`,
						Return: `#96544f`,
					},
				},
				Function: func(color any, percent float64) (string, error) {
					if c, err := colorutil.Desaturate(color, int(percent)); err == nil {
						return c.String(), nil
					} else {
						return ``, err
					}
				},
			}, {
				Name:    `mix`,
				Summary: `Mix two colors together, producing a third.`,
				Arguments: []funcArg{
					{
						Name:        `first`,
						Type:        `string`,
						Description: `The first color to operate on.  Can be specified as rgb(), rgba(), hsl(), hsla(), hsv(), hsva(), "#RRGGBB", or "#RRGGBBAA".`,
					}, {
						Name:        `second`,
						Type:        `string`,
						Description: `The second color to operate on.  Can be specified as rgb(), rgba(), hsl(), hsla(), hsv(), hsva(), "#RRGGBB", or "#RRGGBBAA".`,
					}, {
						Name:        `weight`,
						Type:        `number`,
						Optional:    true,
						Description: `A weight value [0.0, 1.0] specifying how much of the first color to include in the mix.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `mix "#ad4038" "#0000ff" 0.8`,
						Return: `#8A3360`,
					},
				},
				Function: func(first any, second any, weight float64) (string, error) {
					if c, err := colorutil.MixN(first, second, weight); err == nil {
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
				Function: func(color any) (string, error) {
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
				Function: func(color any) (string, error) {
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
				Function: func(color any) (string, error) {
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
				Function: func(value any) string {
					var mmh3 = murmur3.New64().Sum([]byte(typeutil.V(value).String()))

					if len(mmh3) >= 3 {
						return `#` + strings.ToUpper(hex.EncodeToString(mmh3[0:3]))
					} else {
						return `#000000`
					}
				},
			}, {
				Name:    `palette`,
				Summary: `Retrieve a color from a named color palette based on index.  See [Color Palettes](#color-palettes) for a description of pre-defined palettes.`,
				Arguments: []funcArg{
					{
						Name: `index`,
						Type: `integer`,
						Description: `The index to retrieve. If the index is larger than the number of colors in the ` +
							`palette, it will wrap-around to the beginning.`,
					}, {
						Name:        `palette`,
						Description: `The name of the palette to use.`,
						Type:        `string`,
						Optional:    true,
						Default:     DefaultColorPalette,
					},
				},
				Examples: []funcExample{
					{
						Code:   `palette 4`,
						Return: palettes[DefaultColorPalette][4],
					}, {
						Code:   `palette -1`,
						Return: palettes[DefaultColorPalette][len(palettes[DefaultColorPalette])-1],
					},
				},
				Function: func(index any, paletteName ...string) (string, error) {
					var name = DefaultColorPalette

					if len(paletteName) > 0 && paletteName[0] != `` {
						name = paletteName[0]
					}

					if palette, ok := palettes[name]; ok {
						var idx = typeutil.Int(index)
						var i = int(idx) % len(palette)

						if i < 0 {
							i = len(palette) + i
						}

						var color = palette[i]
						return color, nil
					} else {
						return ``, fmt.Errorf("unknown color palette %q", name)
					}
				},
			}, {
				Name:    `definePalette`,
				Summary: `Define a new named color palette.`,
				Arguments: []funcArg{
					{
						Name: `name`,
						Type: `string`,
					}, {
						Name:        `colors`,
						Description: `A sequence of colors to add to the palette.`,
						Variadic:    true,
					},
				},
				Examples: []funcExample{
					{
						Code: `definePalette "rgb" "#FF0000" "#00FF00" "#0000FF"`,
					},
				},
				Function: func(name string, colors ...string) error {
					if name != `` {
						palettes[name] = colors
						return nil
					} else {
						return fmt.Errorf("must provide a name for the color palette being defined")
					}
				},
			}, {
				Name:    `palettes`,
				Summary: `Returns the definition of all defined palettes.`,
				Function: func() map[string][]string {
					return palettes
				},
			},
		},
	}
}
