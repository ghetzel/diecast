package diecast

import (
	"encoding/json"
	"fmt"
	"github.com/ghetzel/go-stockutil/stringutil"
	"html/template"
	"math"
	"strings"
	"time"
)

func GetStandardFunctions() template.FuncMap {
	rv := make(template.FuncMap)

	// string processing
	rv[`contains`] = strings.Contains
	rv[`lower`] = strings.ToLower
	rv[`ltrim`] = strings.TrimPrefix
	rv[`replace`] = strings.Replace
	rv[`rtrim`] = strings.TrimSuffix
	rv[`split`] = func(input string, delimiter string, n ...int) []string {
		if len(n) == 0 {
			return strings.Split(input, delimiter)
		} else {
			return strings.SplitN(input, delimiter, n[0])
		}
	}
	rv[`strcount`] = strings.Count
	rv[`titleize`] = strings.Title
	rv[`trim`] = strings.TrimSpace
	rv[`upper`] = strings.ToUpper
	rv[`hasPrefix`] = strings.HasPrefix
	rv[`hasSuffix`] = strings.HasSuffix

	rv[`percent`] = func(value interface{}, args ...interface{}) (string, error) {
		if v, err := stringutil.ConvertToFloat(value); err == nil {
			outOf := 100.0
			format := "%.f"

			if len(args) > 0 {
				if o, err := stringutil.ConvertToFloat(args[0]); err == nil {
					outOf = o
				} else {
					return ``, err
				}
			}

			if len(args) > 1 {
				format = fmt.Sprintf("%v", args[1])
			}

			percent := float64((float64(v) / float64(outOf)) * 100.0)

			return fmt.Sprintf(format, percent), nil
		} else {
			return ``, err
		}
	}

	// encoding
	rv[`jsonify`] = func(value interface{}, indent ...string) (string, error) {
		indentString := ``

		if len(indent) > 0 {
			indentString = indent[0]
		}

		data, err := json.MarshalIndent(value, ``, indentString)
		return string(data[:]), err
	}

	// type handling and conversion
	rv[`isBool`] = stringutil.IsBoolean
	rv[`isInt`] = stringutil.IsInteger
	rv[`isFloat`] = stringutil.IsFloat
	rv[`autotype`] = stringutil.Autotype
	rv[`asStr`] = stringutil.ToString
	rv[`asInt`] = stringutil.ConvertToInteger
	rv[`asFloat`] = stringutil.ConvertToFloat
	rv[`asBool`] = stringutil.ConvertToBool
	rv[`asTime`] = stringutil.ConvertToTime
	rv[`autobyte`] = stringutil.ToByteString

	// time and date formatting
	tmFmt := func(value interface{}, format ...string) (string, error) {
		if v, err := stringutil.ConvertToTime(value); err == nil {
			var tmFormat string

			if len(format) == 0 {
				tmFormat = time.RFC3339
			} else {
				switch format[0] {
				case `kitchen`:
					tmFormat = time.Kitchen
				case `rfc3339`:
					tmFormat = time.RFC3339
				case `rfc3339ns`:
					tmFormat = time.RFC3339Nano
				case `rfc822`:
					tmFormat = time.RFC822
				case `rfc822z`:
					tmFormat = time.RFC822Z
				case `epoch`:
					return fmt.Sprintf("%d", v.Unix()), nil
				case `epoch-ms`:
					return fmt.Sprintf("%d", int64(v.UnixNano()/1000000)), nil
				case `epoch-us`:
					return fmt.Sprintf("%d", int64(v.UnixNano()/1000)), nil
				case `day`:
					tmFormat = `Monday`
				case `slash`:
					tmFormat = `01/02/2006`
				case `slash-dmy`:
					tmFormat = `02/01/2006`
				case `ymd`:
					tmFormat = `2006-01-02`
				case `ruby`:
					tmFormat = time.RubyDate
				default:
					tmFormat = format[0]
				}
			}

			return v.Format(tmFormat), nil
		} else {
			return ``, err
		}
	}

	rv[`time`] = tmFmt
	rv[`now`] = func(format ...string) (string, error) {
		return tmFmt(time.Now(), format...)
	}

	// numeric/math functions
	calcFn := func(op string, values ...float64) (float64, error) {
		switch len(values) {
		case 0:
			return 0.0, nil
		case 1:
			return values[0], nil
		default:
			out := values[0]

			for _, v := range values[1:] {
				switch op {
				case `+`:
					out += v
				case `-`:
					out -= v
				case `*`:
					out *= v
				case `^`:
					out = math.Pow(out, v)
				case `/`:
					if v == 0.0 {
						return 0, fmt.Errorf("cannot divide by zero")
					}

					out /= v
				case `%`:
					if v == 0.0 {
						return 0, fmt.Errorf("cannot divide by zero")
					}

					out = math.Mod(out, v)
				}
			}

			return out, nil
		}
	}

	rv[`calc`] = calcFn

	rv[`add`] = func(values ...float64) float64 {
		out, _ := calcFn(`+`, values...)
		return out
	}

	rv[`subtract`] = func(values ...float64) float64 {
		out, _ := calcFn(`-`, values...)
		return out
	}

	rv[`multiply`] = func(values ...float64) float64 {
		out, _ := calcFn(`*`, values...)
		return out
	}

	rv[`divide`] = func(values ...float64) (float64, error) {
		return calcFn(`/`, values...)
	}

	rv[`mod`] = func(values ...float64) (float64, error) {
		return calcFn(`%`, values...)
	}

	rv[`pow`] = func(values ...float64) (float64, error) {
		return calcFn(`^`, values...)
	}

	rv[`sequence`] = func(max float64) []int {
		seq := make([]int, int(max))

		for i, _ := range seq {
			seq[i] = i
		}

		return seq
	}

	return rv
}
