package diecast

import (
	"github.com/ghetzel/go-stockutil/convutil"
)

var ConvertRoundToPlaces = 12

func loadStandardFunctionsConvert() funcGroup {
	return funcGroup{
		Name:        `Unit Conversions`,
		Description: `Used to convert numeric values between different unit systems.`,
		Functions: []funcDef{
			{
				Name: `convert`,
				Description: `A generic unit conversion function that allows for units to be ` +
					`specified by value as strings.`,
				Function: convutil.Convert,
			},
		},
	}
}
