package functions

import (
    "html/template"
    "strconv"
    "github.com/shutterstock/go-stockutil/stringutil"
    "github.com/ghetzel/diecast/diecast/util"
)

func GetBaseFunctions() template.FuncMap {
    return template.FuncMap{
        `autosize`: func(input float64, fixTo int) (string, error) {
            check := float64(input)
            i := 1

            for i = 1; i < 9; i++ {
                if check < 1024.0 {
                    break
                }else{
                    check = (check / 1024.0)
                }
            }

            return (strconv.FormatFloat(check, 'f', fixTo, 64) + ` ` + util.SiSuffixes[i-1]), nil
        },
        `length`: func(set []interface{}) int {
            return len(set)
        },
        `str`: func(in ...interface{}) (string, error) {
            if len(in) > 0 {
                if in[0] != nil {
                    return stringutil.ToString(in[0])
                }
            }

            return ``, nil
        },
    }
}