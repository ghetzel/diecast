package diecast

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/rxutil"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/typeutil"
	"github.com/kyokomi/emoji"
)

var emojiCodeMap = emoji.CodeMap()

func loadStandardFunctionsString(funcs FuncMap, server *Server) funcGroup {
	return funcGroup{
		Name: `String Functions`,
		Description: `Used to modify strings in various ways. Whitespace trimming, substring and ` +
			`concatenation, conversion, and find & replace functions can all be found here.`,
		Functions: []funcDef{
			{
				Name:     `contains`,
				Summary:  `Return true of the given string contains another string.`,
				Function: strings.Contains,
				Arguments: []funcArg{
					{
						Name:        `input`,
						Type:        `string`,
						Description: `The string to search within.`,
					}, {
						Name:        `substring`,
						Type:        `string`,
						Description: `The substring to find in the input string.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `contains "Alice met Bob at the store." "store"`,
						Return: `true`,
					},
				},
			}, {
				Name:     `lower`,
				Summary:  `Reformat the given string by changing it into lower case capitalization.`,
				Function: strings.ToLower,
				Arguments: []funcArg{
					{
						Name:        `in`,
						Type:        `string`,
						Description: `The input string to reformat.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `lower "This is a thing"`,
						Return: `this is a thing`,
					},
				},
			}, {
				Name:    `ltrim`,
				Summary: `Return the given string with any leading whitespace removed.`,
				Function: func(in interface{}, str string) string {
					return strings.TrimPrefix(fmt.Sprintf("%v", in), str)
				},
				Arguments: []funcArg{
					{
						Name:        `in`,
						Type:        `string`,
						Description: `The input string to trim.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `trim " Hello   World  "`,
						Return: `Hello  World  `,
					},
				},
			}, {
				Name:     `replace`,
				Summary:  `Replace occurrences of one substring with another string in a given input string.`,
				Function: strings.Replace,
				Arguments: []funcArg{
					{
						Name:        `wholestring`,
						Type:        `string`,
						Description: `The whole string being searched.`,
					}, {
						Name:        `old`,
						Type:        `string`,
						Description: `The old value being sought.`,
					}, {
						Name:        `new`,
						Type:        `string`,
						Description: `The new value that is replacing old.`,
					}, {
						Name:        `count`,
						Type:        `integer`,
						Description: `The number of matches to replace before stopping. If this number is < 0, the all occurrences will be replaced.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `replace "oink oink oink" "oink" "moo" -1`,
						Return: `moo moo moo`,
					}, {
						Code:   `replace "cheese" "e" "o" 2`,
						Return: `choose`,
					},
				},
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
				Arguments: []funcArg{
					{
						Name:        `wholestring`,
						Type:        `string`,
						Description: `The whole string being searched.`,
					}, {
						Name:        `pattern`,
						Type:        `string`,
						Description: `A Golang-compatible regular expression that matches what should be replaced.`,
					}, {
						Name:        `repl`,
						Type:        `string`,
						Description: `The string to replace matches with.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `rxreplace "<b>Hello <i>World</i></b>" "</?[bi]>" "*"`,
						Return: `*Hello *World**`,
					},
				},
			}, {
				Name:    `concat`,
				Summary: `Return the string that results in stringifying and joining all of the given arguments.`,
				Function: func(in ...interface{}) string {
					var out = make([]string, len(in))

					for i, v := range in {
						out[i] = fmt.Sprintf("%v", v)
					}

					return strings.Join(out, ``)
				},
				Arguments: []funcArg{
					{
						Name:        `values`,
						Type:        `any`,
						Description: `One or more values to be stringified and joined together.`,
						Variadic:    true,
					},
				},
				Examples: []funcExample{
					{
						Code:   `concat "There are " 5 " apples, yes it's " true`,
						Return: `There are 5 apples, yes it's true.`,
					},
				},
			}, {
				Name:    `rtrim`,
				Summary: `Return the given string with any trailing whitespace removed.`,
				Function: func(in interface{}, str string) string {
					return strings.TrimSuffix(fmt.Sprintf("%v", in), str)
				},
				Arguments: []funcArg{
					{
						Name:        `in`,
						Type:        `string`,
						Description: `The input string to trim.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `trim " Hello   World  "`,
						Return: ` Hello  World`,
					},
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
				Arguments: []funcArg{
					{
						Name:        `in`,
						Type:        `string`,
						Description: `The string to split into pieces.`,
					}, {
						Name:        `separator`,
						Type:        `string`,
						Description: `The separator on which the input will be split.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `split "this is a sentence."`,
						Return: []string{`this`, `is`, `a`, `sentence.`},
					},
				},
			}, {
				Name: `join`,
				Summary: `Stringify the given array of values and join them together into a string, ` +
					`separated by a given separator string.`,
				Arguments: []funcArg{
					{
						Name:        `input`,
						Type:        `array[any]`,
						Description: `An array of values to stringify and join.`,
					}, {
						Name:        `separator`,
						Type:        `string`,
						Description: `The string used to join all elements of the array together.`,
					}, {
						Name:        `outerDelimiter`,
						Type:        `string`,
						Optional:    true,
						Description: `If given an object, this string will be used to join successive key-value pairs.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `join [1, 2, 3] ","`,
						Return: `1,2,3`,
					}, {
						Code:   `join {"a": 1, "b": 2, "c": 3} "=" "&"`,
						Return: `a=1&b=2&c=3`,
					},
				},
				Function: func(input interface{}, delimiter string, outerDelimiter ...string) string {
					if typeutil.IsMap(input) {
						var od = ``

						if len(outerDelimiter) > 0 {
							od = outerDelimiter[0]
						}

						return maputil.Join(input, delimiter, od)
					} else {
						var inStr = sliceutil.Stringify(input)
						return strings.Join(inStr, delimiter)
					}
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
					var str = stringutil.Camelize(s)

					for i, v := range str {
						return string(unicode.ToLower(v)) + str[i+1:]
					}

					return str
				},
				Arguments: []funcArg{
					{
						Name:        `in`,
						Type:        `string`,
						Description: `The input string to reformat.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `camelize "This is a thing"`,
						Return: `thisIsAThing`,
					},
				},
			}, {
				Name:     `pascalize`,
				Summary:  `Reformat the given string by changing it into PascalCase capitalization.`,
				Function: stringutil.Camelize,
				Arguments: []funcArg{
					{
						Name:        `in`,
						Type:        `string`,
						Description: `The input string to reformat.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `pascalize "This is a thing"`,
						Return: `ThisIsAThing`,
					},
				},
			}, {
				Name:     `underscore`,
				Summary:  `Reformat the given string by changing it into \_underscorecase\_ capitalization (also known as snake\_case).`,
				Function: stringutil.Underscore,
				Arguments: []funcArg{
					{
						Name:        `in`,
						Type:        `string`,
						Description: `The input string to reformat.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `underscore "This is a thing"`,
						Return: `this_is_a_thing`,
					},
				},
			}, {
				Name:     `hyphenate`,
				Summary:  `Reformat the given string by changing it into hyphen-case capitalization.`,
				Function: stringutil.Hyphenate,
				Arguments: []funcArg{
					{
						Name:        `in`,
						Type:        `string`,
						Description: `The input string to reformat.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `hyphenate "This is a thing"`,
						Return: `this-is-a-thing`,
					},
				},
			}, {
				Name:    `trim`,
				Summary: `Return the given string with any leading and trailing whitespace or characters removed.`,
				Arguments: []funcArg{
					{
						Name:        `in`,
						Type:        `string`,
						Description: `The input string to trim.`,
					}, {
						Name:        `characters`,
						Type:        `string`,
						Description: `A sequence of characters to trim from the string.`,
						Optional:    true,
					},
				},
				Examples: []funcExample{
					{
						Code:   `trim " Hello   World  "`,
						Return: `Hello  World`,
					}, {
						Code:   `trim "'hello world'" "'"`,
						Return: `hello world`,
					},
				},
				Function: func(in interface{}, cuts ...string) string {
					var cutset = ``

					if len(cuts) > 0 {
						cutset = strings.Join(cuts, ``)
					}

					if cutset == `` {
						return strings.TrimSpace(typeutil.String(in))
					} else {
						return strings.Trim(typeutil.String(in), cutset)
					}
				},
			}, {
				Name:     `upper`,
				Summary:  `Reformat the given string by changing it into UPPER CASE capitalization.`,
				Function: strings.ToUpper,
				Arguments: []funcArg{
					{
						Name:        `in`,
						Type:        `string`,
						Description: `The input string to reformat.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `upper "This is a thing`,
						Return: `THIS IS A THING`,
					},
				},
			}, {
				Name:     `hasPrefix`,
				Summary:  `Return true if the given string begins with the given prefix.`,
				Function: strings.HasPrefix,
				Arguments: []funcArg{
					{
						Name:        `in`,
						Type:        `string`,
						Description: `The input string to test.`,
					}, {
						Name:        `prefix`,
						Type:        `string`,
						Description: `The prefix to test for the presence of.`,
					},
				},
			}, {
				Name:     `hasSuffix`,
				Summary:  `Return true if the given string ends with the given suffix.`,
				Function: strings.HasSuffix,
				Arguments: []funcArg{
					{
						Name:        `in`,
						Type:        `string`,
						Description: `The input string to test.`,
					}, {
						Name:        `suffix`,
						Type:        `string`,
						Description: `The suffix to test for the presence of.`,
					},
				},
			}, {
				Name:    `surroundedBy`,
				Summary: `Return whether the given string is begins with a specific prefix _and_ ends with a specific suffix.`,
				Arguments: []funcArg{
					{
						Name:        `input`,
						Type:        `string`,
						Description: `The string to test.`,
					}, {
						Name:        `prefix`,
						Type:        `string`,
						Description: `A string to prepend to the given input string.`,
					}, {
						Name:        `suffix`,
						Type:        `string`,
						Description: `A string to append to the given input string.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `surroundedBy "<table>" "<" ">"`,
						Return: true,
					},
				},
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
						var outOf = 100.0
						var format = "%.f"

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

						var percent = float64((float64(v) / float64(outOf)) * 100.0)

						return fmt.Sprintf(format, percent), nil
					} else {
						return ``, err
					}
				},
				Arguments: []funcArg{
					{
						Name:        `value`,
						Type:        `number`,
						Description: `The value you wish to express as a percentage.`,
					}, {
						Name:        `whole`,
						Type:        `number`,
						Description: `The number that represents 100%.`,
					}, {
						Name:        `format`,
						Type:        `string`,
						Optional:    true,
						Default:     `%.f`,
						Description: `The printf format string used for rounding and truncating the converted number.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `percent 99`,
						Return: `99`,
					}, {
						Code:   `percent 3.3 10`,
						Return: `33`,
					}, {
						Code:   `percent 3.33 10 "%.3f"`,
						Return: `33.300`,
					},
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
				Arguments: []funcArg{
					{
						Name:        `bytes`,
						Type:        `number`,
						Description: `A number representing the value to format, in bytes.`,
					}, {
						Name:        `format`,
						Type:        `string`,
						Description: `A printf-style format string used to represent the output number.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `autobyte 2490368 "%.2f"`,
						Return: `2.38MB`,
					}, {
						Code:   `autobyte 9876543210 "%.0f "`,
						Return: `9 GB`,
					},
				},
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
					return stringutil.ElideWords(fmt.Sprintf("%v", in), wordcount)
				},
				Arguments: []funcArg{
					{
						Name:        `input`,
						Type:        `string`,
						Description: `The string to (possibly) truncate.`,
					}, {
						Name:        `wordcount`,
						Type:        `integer`,
						Description: `The maximum number of words that can appear in a string before it is truncated.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `elideWords "This is a sentence that contains eight words." 5`,
						Return: `This is a sentence that`,
					}, {
						Code:   `elideWords "Hello world" 10`,
						Return: `Hello world`,
					},
				},
			}, {
				Name:    `elide`,
				Summary: `Takes an input string and ensures it is no longer than a given number of characters.`,
				Function: func(in interface{}, charcount int) string {
					var inS = fmt.Sprintf("%v", in)

					if len(inS) > charcount {
						inS = inS[0:charcount]
					}

					if match := rxutil.Match(`(\W*\s+[\w\.\(\)\[\]\{\}]{0,16})$`, inS); match != nil {
						inS = match.ReplaceGroup(1, ``)
					}

					return inS
				},
				Arguments: []funcArg{
					{
						Name:        `input`,
						Type:        `string`,
						Description: `The string to (possibly) truncate.`,
					}, {
						Name:        `charcount`,
						Type:        `integer`,
						Description: `The maximum number of characters that can appear in a string before it is truncated.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `elide "This is a sentence that contains fifty characters." 18`,
						Return: `This is a sentence`,
					}, {
						Code:   `elide "hello." 16`,
						Return: `hello.`,
					},
				},
			}, {
				Name:    `emoji`,
				Summary: `Expand the given named emoji into its literal Unicode character.`,
				Arguments: []funcArg{
					{
						Name:        `emoji`,
						Type:        `string`,
						Description: `The common name of the emoji to return, with or without surrounding colons (:).`,
					}, {
						Name:        `fallback`,
						Type:        `string`,
						Description: `What to return if the named emoji is not found.`,
					},
				},
				Function: func(in interface{}, fallbacks ...string) string {
					var name = typeutil.String(in)
					name = strings.TrimPrefix(name, `:`)
					name = strings.TrimSuffix(name, `:`)
					name = `:` + stringutil.Underscore(name) + `:`
					name = strings.ToLower(name)

					if emj, ok := emojiCodeMap[name]; ok {
						return emj
					} else if len(fallbacks) > 0 {
						return fallbacks[0]
					} else {
						return ``
					}
				},
				Examples: []funcExample{
					{
						Code:   `emoji ":thinking_face:"`,
						Return: "\U0001f914",
					}, {
						Code:   `emoji ":not_a_real_emoji:" "nope"`,
						Return: `nope`,
					},
				},
			}, {
				Name:    `section`,
				Summary: `Takes an input string, splits it on a given regular expression, and returns the nth field.`,
				Function: func(in interface{}, field int, rx ...string) (string, error) {
					var rxSplit = rxutil.Whitespace
					var input = typeutil.String(in)

					if len(rx) > 0 && rx[0] != `` {
						if x, err := regexp.Compile(rx[0]); err == nil {
							rxSplit = x
						} else {
							return ``, err
						}
					}

					if sections := rxSplit.Split(input, -1); field < len(sections) {
						return sections[field], nil
					} else {
						return ``, nil
					}

				},
				Arguments: []funcArg{
					{
						Name:        `input`,
						Type:        `string`,
						Description: `The string to retrieve the field from.`,
					}, {
						Name:        `field`,
						Type:        `integer`,
						Description: `The number of the field to retrieve after splitting input.`,
					}, {
						Name:        `split`,
						Type:        `string`,
						Optional:    true,
						Description: `A regular expression to use when splitting the string.`,
						Default:     rxutil.Whitespace,
					},
				},
				Examples: []funcExample{
					{
						Code:   `elide "This is a sentence that contains fifty characters." 18`,
						Return: `This is a sentence`,
					}, {
						Code:   `elide "hello." 16`,
						Return: `hello.`,
					},
				},
			}, {
				Name:    `longestString`,
				Summary: `Return the string in the given array that is longest.`,
				Arguments: []funcArg{
					{
						Name:        `array`,
						Type:        `string`,
						Description: `The array of strings to scan.`,
					},
				},
				Function: func(in interface{}) string {
					var largest string

					for _, item := range sliceutil.Stringify(in) {
						if len(item) > len(largest) {
							largest = item
						}
					}

					return largest
				},
			},
		},
	}
}
