package pongo

import (
    "strconv"
    "github.com/flosch/pongo2"
    "github.com/shutterstock/go-stockutil/stringutil"
    "github.com/ghetzel/diecast/diecast/util"
)

func GetBaseFunctions() map[string]pongo2.FilterFunction {
    return map[string]pongo2.FilterFunction{
        `autosize`: func(input *pongo2.Value, fixTo *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
            check := input.Float()
            i := 1

            for i = 1; i < 9; i++ {
                if check < 1024.0 {
                    break
                }else{
                    check = (check / 1024.0)
                }
            }

            return pongo2.AsValue((strconv.FormatFloat(check, 'f', fixTo.Integer(), 64) + ` ` + util.SiSuffixes[i-1])), nil
        },
        `less`: func(first *pongo2.Value, second *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
            return pongo2.AsValue(first.Float() - second.Float()), nil
        },
        `str`: func(input *pongo2.Value, _ *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
            if input.Len() > 0 {
                if v, err := stringutil.ToString(input.Interface()); err == nil {
                    return pongo2.AsValue(v), nil
                }
            }

            return pongo2.AsValue(``), nil
        },
    }
}