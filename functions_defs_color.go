package diecast

import "github.com/ghetzel/go-stockutil/colorutil"

func loadStandardFunctionsColor(rv FuncMap) {
	// fn lighten: Lighten the given color by a percent.
	rv[`lighten`] = func(color interface{}, percent float64) (string, error) {
		if c, err := colorutil.Lighten(color, int(percent)); err == nil {
			return c.String(), nil
		} else {
			return ``, err
		}
	}

	// fn darken: Darken the given color by a percent.
	rv[`darken`] = func(color interface{}, percent float64) (string, error) {
		if c, err := colorutil.Darken(color, int(percent)); err == nil {
			return c.String(), nil
		} else {
			return ``, err
		}
	}

	// fn colorToHex: Convert the given color to a "#RRGGBB" or "#RRGGBBAA" color specification.
	rv[`colorToHex`] = func(color interface{}) (string, error) {
		if c, err := colorutil.Parse(color); err == nil {
			return c.String(), nil
		} else {
			return ``, err
		}
	}

	// fn colorToRGB: Convert the given color to an "rgb()" or "rgba()" color specification.
	rv[`colorToRGB`] = func(color interface{}) (string, error) {
		if c, err := colorutil.Parse(color); err == nil {
			return c.StringRGBA(), nil
		} else {
			return ``, err
		}
	}

	// fn colorToHSL: Convert the given color to an "hsl()" or "hsla()" color specification.
	rv[`colorToHSL`] = func(color interface{}) (string, error) {
		if c, err := colorutil.Parse(color); err == nil {
			return c.StringHSLA(), nil
		} else {
			return ``, err
		}
	}
}
