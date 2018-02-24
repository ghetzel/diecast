package diecast

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/stringutil"
)

func loadStandardFunctionsString(rv FuncMap) {
	// fn contains: Return whether a string *s* contains *substr*.
	rv[`contains`] = strings.Contains

	// fn lower: Return a copy of string *s* with all Unicode letters mapped to their lower case.
	rv[`lower`] = strings.ToLower

	// fn ltrim: Return a copy of string *s* with the leading *prefix* removed.
	rv[`ltrim`] = strings.TrimPrefix

	// fn replace: Return a copy of *s* with occurrences of *old* replaced with *new*, up to *n* times.
	rv[`replace`] = strings.Replace

	// fn rxreplace: Return a copy of *s* with all occurrences of *pattern* replaced with *repl*.
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

	rv[`concat`] = func(in ...interface{}) string {
		out := make([]string, len(in))

		for i, v := range in {
			out[i] = fmt.Sprintf("%v", v)
		}

		return strings.Join(out, ``)
	}

	// fn rtrim: Return a copy of string *s* with the trailing *suffix* removed.
	rv[`rtrim`] = strings.TrimSuffix

	// fn split: Return a string array of elements resulting from *s* being split by *delimiter*,
	//           up to *n* times (if specified).
	rv[`split`] = func(input string, delimiter string, n ...int) []string {
		if len(n) == 0 {
			return strings.Split(input, delimiter)
		} else {
			return strings.SplitN(input, delimiter, n[0])
		}
	}

	// fn join: Join the *input* array on *delimiter* and return a string.
	rv[`join`] = func(input interface{}, delimiter string) string {
		inStr := sliceutil.Stringify(input)
		return strings.Join(inStr, delimiter)
	}

	// fn strcount: Count *s* for the number of non-overlapping instances of *substr*.
	rv[`strcount`] = strings.Count

	// fn titleize: Return a copy of *s* with all Unicode letters that begin words mapped to their title case.
	rv[`titleize`] = strings.Title

	// fn camelize: Return a copy of *s* transformed into CamelCase.
	rv[`camelize`] = stringutil.Camelize

	// fn underscore: Return a copy of *s* transformed into snake_case.
	rv[`underscore`] = stringutil.Underscore

	// fn trim: Return a copy of *s* with all leading and trailing whitespace characters removed.
	rv[`trim`] = strings.TrimSpace

	// fn upper: Return a copy of *s* with all letters capitalized.
	rv[`upper`] = strings.ToUpper

	// fn hasPrefix: Return whether string *s* has the given *prefix*.
	rv[`hasPrefix`] = strings.HasPrefix

	// fn hasSuffix: Return whether string *s* has the given *suffix*.
	rv[`hasSuffix`] = strings.HasSuffix

	// fn surroundedBy: Return whether string *s* starts with *prefix* and ends with *suffix*.
	rv[`surroundedBy`] = func(value interface{}, prefix string, suffix string) bool {
		if v := fmt.Sprintf("%v", value); strings.HasPrefix(v, prefix) && strings.HasSuffix(v, suffix) {
			return true
		}

		return false
	}

	// fn percent: Return the given floating point *value* as a percentage of *n*, or 100.0 if
	//             *n* is not specified.
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

	// fn autobyte: Attempt to convert the given *bytesize* number to a string representation of the value in bytes.
	rv[`autobyte`] = stringutil.ToByteString

	// fn thousandify: Return a copy of *value* separated by *sep* (or comma by default) every three decimal places.
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
}
