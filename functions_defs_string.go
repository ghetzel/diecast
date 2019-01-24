package diecast

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/ghetzel/go-stockutil/rxutil"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/stringutil"
)

func loadStandardFunctionsString(rv FuncMap) {
	return funcGroup{
		Name: `String`,
		Description: `Used to modify strings in various ways. Whitespace trimming, substring and ` +
			`concatenation, conversion, and find & replace functions can all be found here.`,
		Functions: []funcDef{
			{
				Name:     `contains`,
				Summary:  `Return true of the given string contains another string.`,
				Function: strings.Contains,
			}, {
				Name:     `lower`,
				Summary:  `Reformat the given string by changing it into lower case capitalization.`,
				Function: strings.ToLower,
			}, {
				Name:    `ltrim`,
				Summary: `Return the given string with any leading whitespace removed.`,
				Function: func(in interface{}, str string) string {
					return strings.TrimPrefix(fmt.Sprintf("%v", in), str)
				},
			}, {
				Name:     `replace`,
				Summary:  `Replace occurrences of one substring with another string in a given input string.`,
				Function: strings.Replace,
			}, {
				Name:    `rxreplace`,
				Summary: `Return the given string with all substrings matching the given regular expression replaced with another string.`,
				Function: func(in interface{}, pattern string, repl string) (string, error) {
					if inS, err := stringutil.ToString(in); err == nil {
						if rx, err := regexp.Compile(pattern); err == nil {
							return rx.ReplaceAllString(inS, repl), nil
						} else {
							return ``, err
						}
					} else {
						return ``, err
					}
				},
			}, {
				Name:    `concat`,
				Summary: `Return the string that results in stringifying and joining all of the given arguments.`,
				Function: func(in ...interface{}) string {
					out := make([]string, len(in))

					for i, v := range in {
						out[i] = fmt.Sprintf("%v", v)
					}

					return strings.Join(out, ``)
				},
			}, {
				Name:    `rtrim`,
				Summary: `Return the given string with any trailing whitespace removed.`,
				Function: func(in interface{}, str string) string {
					return strings.TrimSuffix(fmt.Sprintf("%v", in), str)
				},
			}, {
				Name:    `split`,
				Summary: `Split a given string into an array of strings by a given separator.`,
				Function: func(input string, delimiter string, n ...int) []string {
					if len(n) == 0 {
						return strings.Split(input, delimiter)
					} else {
						return strings.SplitN(input, delimiter, n[0])
					}
				},
			}, {
				Name: `join`,
				Summary: `Stringify the given array of values and join them together into a string, ` +
					`separated by a given separator string.`,
				Function: func(input interface{}, delimiter string) string {
					inStr := sliceutil.Stringify(input)
					return strings.Join(inStr, delimiter)
				},
			}, {
				Name: `strcount`,
				Summary: `Count counts the number of non-overlapping instances of a substring. If ` +
					`the given substring is empty, then this returns the length of the string plus one.`,
				Function: strings.Count,
			}, {
				Name:     `titleize`,
				Summary:  `Reformat the given string by changing it into Title Case capitalization.`,
				Function: strings.Title,
			}, {
				Name:    `camelize`,
				Summary: `Reformat the given string by changing it into camelCase capitalization.`,
				Function: func(s interface{}) string {
					str := stringutil.Camelize(s)

					for i, v := range str {
						return string(unicode.ToLower(v)) + str[i+1:]
					}

					return str
				},
			}, {
				Name:     `pascalize`,
				Summary:  `Reformat the given string by changing it into PascalCase capitalization.`,
				Function: stringutil.Camelize,
			}, {
				Name:     `underscore`,
				Summary:  `Reformat the given string by changing it into \_underscorecase\_ capitalization (also known as snake\_case).`,
				Function: stringutil.Underscore,
			}, {
				Name:     `hyphenate`,
				Summary:  `Reformat the given string by changing it into hyphen-case capitalization.`,
				Function: stringutil.Hyphenate,
			}, {
				Name:     `trim`,
				Summary:  `Return the given string with any leading and trailing whitespace removed.`,
				Function: strings.TrimSpace,
			}, {
				Name:     `upper`,
				Summary:  `Reformat the given string by changing it into UPPER CASE capitalization.`,
				Function: strings.ToUpper,
			}, {
				Name:     `hasPrefix`,
				Summary:  `Return true if the given string begins with the given prefix.`,
				Function: strings.HasPrefix,
			}, {
				Name:     `hasSuffix`,
				Summary:  `Return true if the given string ends with the given suffix.`,
				Function: strings.HasSuffix,
			}, {
				Name:    `surroundedBy`,
				Summary: `Return whether the given string is begins with a specific prefix _and_ ends with a specific suffix.`,
				Function: func(value interface{}, prefix string, suffix string) bool {
					if v := fmt.Sprintf("%v", value); strings.HasPrefix(v, prefix) && strings.HasSuffix(v, suffix) {
						return true
					}

					return false
				},
			}, {
				Name:    `percent`,
				Summary: `Takes an integer or decimal value and returns it formatted as a percentage.`,
				Function: func(value interface{}, args ...interface{}) (string, error) {
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
				},
			}, {
				Name: `autobyte`,
				Summary: `Attempt to convert the given number to a string representation of the ` +
					`value interpreted as bytes. The unit will be automatically determined as the ` +
					`closest one that produces a value less than 1024 in that unit. The second ` +
					`argument is a printf-style format string that is used when the converted number ` +
					`is being stringified. By specifying precision and leading digit values to the %f ` +
					`format token, you can control how many decimal places are in the resulting output.`,
				Function: stringutil.ToByteString,
			}, {
				Name: `thousandify`,
				Summary: `Take a number and reformat it to be more readable by adding a separator ` +
					`between every three successive places.`,
				Function: func(value interface{}, sepDec ...string) string {
					var separator string
					var decimal string

					if len(sepDec) > 0 {
						separator = sepDec[0]
					}

					if len(sepDec) > 1 {
						decimal = sepDec[1]
					}

					return stringutil.Thousandify(value, separator, decimal)
				},
			}, {
				Name: `splitWords`,
				Summary: `Detect word boundaries in a given string and split that string into an ` +
					`array where each element is a word.`,
				Function: func(in interface{}) []string {
					return stringutil.SplitWords(fmt.Sprintf("%v", in))
				},
			}, {
				Name: `elideWords`,
				Summary: `Takes an input string and counts the number of words in it. If that number ` +
					`exceeds a given count, the string will be truncated to be equal to that number of words.`,
				Function: func(in interface{}, wordcount int) string {
					return stringutil.ElideWords(fmt.Sprintf("%v", in), uint(wordcount))
				},
			}, {
				Name:    `elide`,
				Summary: `Takes an input string and ensures it is no longer than a given number of characters.`,
				Function: func(in interface{}, charcount int) string {
					inS := fmt.Sprintf("%v", in)

					if len(inS) > charcount {
						inS = inS[0:charcount]
					}

					if match := rxutil.Match(`(\W*\s+[\w\.\(\)\[\]\{\}]{0,16})$`, inS); match != nil {
						inS = match.ReplaceGroup(1, ``)
					}

					return inS
				},
			},
		},
	}
}
