package diecast

import (
	"fmt"
	htmlmain "html"
	"regexp"
	"strings"
	"unicode"

	"github.com/ghetzel/go-stockutil/rxutil"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/stringutil"
	strip "github.com/grokify/html-strip-tags-go"
)

func loadStandardFunctionsString(rv FuncMap) {
	// fn contains: Return whether a string *s* contains *substr*.
	rv[`contains`] = strings.Contains

	// fn lower: Return a copy of string *s* with all Unicode letters mapped to their lower case.
	rv[`lower`] = strings.ToLower

	// fn ltrim: Return a copy of string *s* with the leading *prefix* removed.
	rv[`ltrim`] = func(in interface{}, str string) string {
		return strings.TrimPrefix(fmt.Sprintf("%v", in), str)
	}

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
	rv[`rtrim`] = func(in interface{}, str string) string {
		return strings.TrimSuffix(fmt.Sprintf("%v", in), str)
	}

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

	// fn camelize: Return a copy of *s* transformed into camelCase.
	rv[`camelize`] = func(s interface{}) string {
		str := stringutil.Camelize(s)

		for i, v := range str {
			return string(unicode.ToLower(v)) + str[i+1:]
		}

		return str
	}

	// fn pascalize: Return a copy of *s* transformed into PascalCase.
	rv[`pascalize`] = stringutil.Camelize

	// fn underscore: Return a copy of *s* transformed into snake_case.
	rv[`underscore`] = stringutil.Underscore

	// fn hyphenate: Return a copy of *s* transformed into hyphen-case.
	rv[`hyphenate`] = stringutil.Hyphenate

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

	rv[`splitWords`] = func(in interface{}) []string {
		return stringutil.SplitWords(fmt.Sprintf("%v", in))
	}

	rv[`elideWords`] = func(in interface{}, wordcount int) string {
		return stringutil.ElideWords(fmt.Sprintf("%v", in), uint(wordcount))
	}

	// fn elide: Truncates the given *text* in a word-aware manner to the given number of characters.
	rv[`elide`] = func(in interface{}, charcount int) string {
		inS := fmt.Sprintf("%v", in)

		if len(inS) > charcount {
			inS = inS[0:charcount]
		}

		if match := rxutil.Match(`(\W*\s+[\w\.\(\)\[\]\{\}]{0,16})$`, inS); match != nil {
			inS = match.ReplaceGroup(1, ``)
		}

		return inS
	}

	// fn stripHtml: strips HTML tags from the given *input* text, leaving the text content behind.
	rv[`stripHtml`] = func(in interface{}) string {
		stripped := strip.StripTags(fmt.Sprintf("%v", in))
		stripped = htmlmain.UnescapeString(stripped)
		return stripped
	}
}
