package diecast

import (
	"crypto/rand"
	"encoding/base32"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/typeutil"
	"github.com/jbenet/go-base58"
	"github.com/microcosm-cc/bluemonday"
	"github.com/montanaflynn/stats"
	"github.com/russross/blackfriday"
	"github.com/satori/go.uuid"
	"github.com/spaolacci/murmur3"
)

var Base32Alphabet = base32.NewEncoding(`abcdefghijklmnopqrstuvwxyz234567`)

type statsUnary func(stats.Float64Data) (float64, error)

func MinNonZero(data stats.Float64Data) (float64, error) {
	for i, v := range data {
		if v == 0 {
			data = append(data[:i], data[i+1:]...)
		}
	}

	return stats.Min(data)
}

func GetStandardFunctions() template.FuncMap {
	rv := make(template.FuncMap)

	// string processing
	rv[`contains`] = strings.Contains
	rv[`lower`] = strings.ToLower
	rv[`ltrim`] = strings.TrimPrefix
	rv[`replace`] = strings.Replace
	rv[`rxreplace`] = func(in interface{}, pattern string, repl string) (string, error) {
		if inS, err := stringutil.ToString(in); err == nil {
			if rx, err := regexp.Compile(pattern); err == nil {
				return rx.ReplaceAllString(inS, repl), nil
			} else {
				return ``, err
			}
		} else {
			return ``, err
		}
	}
	rv[`rtrim`] = strings.TrimSuffix
	rv[`split`] = func(input string, delimiter string, n ...int) []string {
		if len(n) == 0 {
			return strings.Split(input, delimiter)
		} else {
			return strings.SplitN(input, delimiter, n[0])
		}
	}

	rv[`join`] = func(input interface{}, delimiter string) string {
		inStr := sliceutil.Stringify(input)
		return strings.Join(inStr, delimiter)
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

	rv[`markdown`] = func(value interface{}) (string, error) {
		input := fmt.Sprintf("%v", value)
		output := blackfriday.MarkdownCommon([]byte(input[:]))
		output = bluemonday.UGCPolicy().SanitizeBytes(output)

		return string(output[:]), nil
	}

	// type handling and conversion
	rv[`isBool`] = stringutil.IsBoolean
	rv[`isInt`] = stringutil.IsInteger
	rv[`isFloat`] = stringutil.IsFloat
	rv[`isZero`] = typeutil.IsZero
	rv[`isEmpty`] = typeutil.IsEmpty
	rv[`autotype`] = stringutil.Autotype
	rv[`asStr`] = stringutil.ToString
	rv[`asInt`] = func(value interface{}) (int64, error) {
		if v, err := stringutil.ConvertToFloat(value); err == nil {
			return int64(v), nil
		} else {
			return 0, err
		}
	}

	rv[`asFloat`] = stringutil.ConvertToFloat
	rv[`asBool`] = stringutil.ConvertToBool
	rv[`asTime`] = stringutil.ConvertToTime
	rv[`autobyte`] = stringutil.ToByteString
	rv[`thousandify`] = func(value interface{}, sepDec ...string) string {
		var separator string
		var decimal string

		if len(sepDec) > 0 {
			separator = sepDec[0]
		}

		if len(sepDec) > 1 {
			decimal = sepDec[1]
		}

		return stringutil.Thousandify(value, separator, decimal)
	}

	// time and date formatting
	tmFmt := func(value interface{}, format ...string) (string, error) {
		if v, err := stringutil.ConvertToTime(value); err == nil {
			var tmFormat string
			var formatName string

			if len(format) == 0 {
				tmFormat = time.RFC3339
			} else {
				formatName = format[0]

				switch formatName {
				case `kitchen`:
					tmFormat = time.Kitchen
				case `timer`:
					tmFormat = `15:04:05`
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
					tmFormat = formatName
				}
			}

			vStr := v.Format(tmFormat)

			if formatName == `timer` {
				if len(strings.Split(vStr, `:`)) == 3 {
					vStr = strings.TrimPrefix(vStr, `00:`)
				}
			}

			return vStr, nil
		} else {
			return ``, err
		}
	}

	rv[`time`] = tmFmt
	rv[`now`] = func(format ...string) (string, error) {
		return tmFmt(time.Now(), format...)
	}

	rv[`duration`] = func(value interface{}, unit string, formats ...string) (string, error) {
		if v, err := stringutil.ConvertToInteger(value); err == nil {
			duration := time.Duration(v)
			format := `timer`

			if len(formats) > 0 {
				format = formats[0]
			}

			switch unit {
			case `ns`, ``:
				break
			case `us`:
				duration = duration * time.Microsecond
			case `ms`:
				duration = duration * time.Millisecond
			case `s`:
				duration = duration * time.Second
			case `m`:
				duration = duration * time.Minute
			case `h`:
				duration = duration * time.Hour
			case `d`:
				duration = duration * time.Hour * 24
			case `y`:
				duration = duration * time.Hour * 24 * 365
			default:
				return ``, fmt.Errorf("Unrecognized unit %q", unit)
			}

			basetime := time.Date(0, 0, 0, 0, 0, 0, 0, time.UTC)
			basetime = basetime.Add(duration)

			return tmFmt(basetime, format)
		} else {
			return ``, err
		}
	}

	// random numbers and encoding
	rv[`random`] = func(count int) ([]byte, error) {
		output := make([]byte, count)
		if _, err := rand.Read(output); err == nil {
			return output, nil
		} else {
			return nil, err
		}
	}

	rv[`uuid`] = func() string {
		return uuid.NewV4().String()
	}

	rv[`uuidRaw`] = func() []byte {
		return uuid.NewV4().Bytes()
	}

	rv[`base32`] = func(input []byte) string {
		return Base32Alphabet.EncodeToString(input)
	}

	rv[`base58`] = func(input []byte) string {
		return base58.Encode(input)
	}

	rv[`base64`] = func(input []byte, encoding ...string) string {
		if len(encoding) == 0 {
			encoding = []string{`standard`}
		}

		switch encoding[0] {
		case `padded`:
			return base64.StdEncoding.EncodeToString(input)
		case `url`:
			return base64.RawURLEncoding.EncodeToString(input)
		case `url-padded`:
			return base64.URLEncoding.EncodeToString(input)
		default:
			return base64.RawStdEncoding.EncodeToString(input)
		}
	}

	rv[`murmur3`] = func(input interface{}) (uint64, error) {
		if v, err := stringutil.ToString(input); err == nil {
			return murmur3.Sum64([]byte(v)), nil
		} else {
			return 0, err
		}
	}

	// numeric/math functions
	calcFn := func(op string, values ...interface{}) (float64, error) {
		valuesF := make([]float64, len(values))

		for i, v := range values {
			if vF, err := stringutil.ConvertToFloat(v); err == nil {
				valuesF[i] = vF
			} else {
				return 0, err
			}
		}

		switch len(valuesF) {
		case 0:
			return 0.0, nil
		case 1:
			return valuesF[0], nil
		default:
			out := valuesF[0]

			for _, v := range valuesF[1:] {
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

	rv[`add`] = func(values ...interface{}) float64 {
		out, _ := calcFn(`+`, values...)
		return out
	}

	rv[`subtract`] = func(values ...interface{}) float64 {
		out, _ := calcFn(`-`, values...)
		return out
	}

	rv[`multiply`] = func(values ...interface{}) float64 {
		out, _ := calcFn(`*`, values...)
		return out
	}

	rv[`divide`] = func(values ...interface{}) (float64, error) {
		return calcFn(`/`, values...)
	}

	rv[`mod`] = func(values ...interface{}) (float64, error) {
		return calcFn(`%`, values...)
	}

	rv[`pow`] = func(values ...interface{}) (float64, error) {
		return calcFn(`^`, values...)
	}

	rv[`sequence`] = func(max interface{}) []int {
		if v, err := stringutil.ConvertToInteger(max); err == nil {
			seq := make([]int, v)

			for i, _ := range seq {
				seq[i] = i
			}

			return seq
		} else {
			return nil
		}
	}

	// numeric aggregation functions
	type statsTplFunc func(in interface{}) (float64, error) // {}

	for fnName, fn := range map[string]statsUnary{
		`maximum`:    stats.Max,
		`mean`:       stats.Mean,
		`median`:     stats.Median,
		`minimum`:    stats.Min,
		`minimum_nz`: MinNonZero,
		`stddev`:     stats.StandardDeviation,
		`sum`:        stats.Sum,
	} {
		rv[fnName] = func(statsFn statsUnary) statsTplFunc {
			return func(in interface{}) (float64, error) {
				var input []float64

				if err := sliceutil.Each(in, func(i int, value interface{}) error {
					if v, err := stringutil.ConvertToFloat(value); err == nil {
						input = append(input, v)
					} else {
						return err
					}

					return nil
				}); err == nil {
					if vv, err := statsFn(stats.Float64Data(input)); err == nil {
						return vv, nil
					} else {
						return 0, nil
					}
				} else {
					return 0, err
				}
			}
		}(fn)
	}

	// simpler, more relaxed comparators
	rv[`eqx`] = typeutil.RelaxedEqual
	rv[`nex`] = func(first interface{}, second interface{}) (bool, error) {
		eq, err := typeutil.RelaxedEqual(first, second)
		return !eq, err
	}

	// set processing
	rv[`asList`] = func(input ...interface{}) []interface{} {
		return input
	}

	rv[`pluck`] = func(input interface{}, key string) []interface{} {
		return maputil.Pluck(input, strings.Split(key, `.`))
	}

	rv[`in`] = func(want interface{}, input []interface{}) bool {
		for _, have := range input {
			if eq, err := typeutil.RelaxedEqual(have, want); err == nil && eq == true {
				return true
			}
		}

		return false
	}

	rv[`indexOf`] = func(slice interface{}, value interface{}) (index int) {
		index = -1

		if typeutil.IsArray(slice) {
			sliceutil.Each(slice, func(i int, v interface{}) error {
				if eq, err := typeutil.RelaxedEqual(v, value); err == nil && eq == true {
					index = i
					return sliceutil.Stop
				} else {
					return nil
				}
			})
		}

		return
	}

	rv[`uniq`] = func(slice interface{}) []interface{} {
		return sliceutil.Unique(slice)
	}

	rv[`compact`] = func(slice []interface{}) []interface{} {
		return sliceutil.Compact(slice)
	}

	rv[`first`] = func(slice interface{}) (out interface{}, err error) {
		err = sliceutil.Each(slice, func(i int, value interface{}) error {
			out = value
			return sliceutil.Stop
		})

		return
	}

	rv[`last`] = func(slice interface{}) (out interface{}, err error) {
		err = sliceutil.Each(slice, func(i int, value interface{}) error {
			out = value
			return nil
		})

		return
	}

	commonses := func(slice interface{}, cmp string) (interface{}, error) {
		counts := make(map[interface{}]int)

		if err := sliceutil.Each(slice, func(i int, value interface{}) error {
			if c, ok := counts[value]; ok {
				counts[value] = c + 1
			} else {
				counts[value] = 1
			}

			return nil
		}); err == nil {
			var out interface{}
			var threshold int

			for value, count := range counts {
				if out == nil {
					out = value
				}

				switch cmp {
				case `most`:
					if count > threshold {
						out = value
						threshold = count
					}
				case `least`:
					if count < threshold {
						out = value
						threshold = count
					}
				default:
					return nil, fmt.Errorf("Unknown comparator %q", cmp)
				}
			}

			return out, nil
		} else {
			return nil, err
		}
	}

	rv[`mostcommon`] = func(slice interface{}) (interface{}, error) {
		return commonses(slice, `most`)
	}

	rv[`leastcommon`] = func(slice interface{}) (interface{}, error) {
		return commonses(slice, `least`)
	}

	rv[`stringify`] = func(slice interface{}) []string {
		return sliceutil.Stringify(slice)
	}

	return rv
}
