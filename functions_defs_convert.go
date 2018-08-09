package diecast

import (
	"github.com/ghetzel/go-stockutil/convutil"
)

var ConvertRoundToPlaces = 12

func loadStandardFunctionsConvert(rv FuncMap) {
	rv[`convert`] = convutil.Convert
}
