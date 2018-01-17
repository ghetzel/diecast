package diecast

import (
	"bytes"
	"crypto/rand"
	"encoding/base32"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"mime"
	"os"
	"path"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/pathutil"
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

type fileInfo struct {
	os.FileInfo
}

func (self *fileInfo) String() string {
	return self.Name()
}

type statsUnary func(stats.Float64Data) (float64, error)

func MinNonZero(data stats.Float64Data) (float64, error) {
	for i, v := range data {
		if v == 0 {
			data = append(data[:i], data[i+1:]...)
		}
	}

	return stats.Min(data)
}

func GetStandardFunctions() FuncMap {
	rv := make(FuncMap)

	// String Processing
	// ---------------------------------------------------------------------------------------------

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

	// File Pathname Handling
	// ---------------------------------------------------------------------------------------------

	// fn basename: Return the filename component of the given *path*.
	rv[`basename`] = func(value interface{}) string {
		return path.Base(fmt.Sprintf("%v", value))
	}

	// fn extname: Return the extension component of the given *path* (always prefixed with a dot [.]).
	rv[`extname`] = func(value interface{}) string {
		return path.Ext(fmt.Sprintf("%v", value))
	}

	// fn dirname: Return the directory path component of the given *path*.
	rv[`dirname`] = func(value interface{}) string {
		return path.Dir(fmt.Sprintf("%v", value))
	}

	// fn pathjoin: Return the value of all *values* join on the system path separator.
	rv[`pathjoin`] = func(values ...interface{}) string {
		return path.Join(sliceutil.Stringify(values)...)
	}

	// fn pwd: Return the present working directory
	rv[`pwd`] = os.Getwd

	// fn dir: Return a list of files and directories in *path*, or in the current directory if not specified.
	rv[`dir`] = func(dirs ...string) ([]*fileInfo, error) {
		var dir string
		entries := make([]*fileInfo, 0)

		if len(dirs) == 0 || dirs[0] == `` {
			if wd, err := os.Getwd(); err == nil {
				dir = wd
			} else {
				return nil, err
			}
		} else {
			dir = dirs[0]
		}

		if d, err := pathutil.ExpandUser(dir); err == nil {
			dir = d
		} else {
			return nil, err
		}

		if e, err := ioutil.ReadDir(dir); err == nil {
			for _, info := range e {
				entries = append(entries, &fileInfo{
					FileInfo: info,
				})
			}

			sort.Slice(entries, func(i, j int) bool {
				return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
			})

			return entries, nil
		} else {
			return nil, err
		}
	}

	// fn mimetype: Returns a best guess MIME type for the given filename
	rv[`mimetype`] = func(filename string) string {
		mime, _ := stringutil.SplitPair(mime.TypeByExtension(path.Ext(filename)), `;`)
		return strings.TrimSpace(mime)
	}

	// fn mimeparams: Returns the parameters portion of the MIME type of the given filename
	rv[`mimeparams`] = func(filename string) map[string]interface{} {
		_, params := stringutil.SplitPair(mime.TypeByExtension(path.Ext(filename)), `;`)
		rv := make(map[string]interface{})

		for _, paramPair := range strings.Split(params, `;`) {
			key, value := stringutil.SplitPair(paramPair, `=`)
			rv[key] = stringutil.Autotype(value)
		}

		return rv
	}

	// Encoding
	// ---------------------------------------------------------------------------------------------

	// fn jsonify: Encode the given *value* as a JSON string, optionally using *indent* to pretty
	//             format the output.
	rv[`jsonify`] = func(value interface{}, indent ...string) (string, error) {
		indentString := ``

		if len(indent) > 0 {
			indentString = indent[0]
		}

		data, err := json.MarshalIndent(value, ``, indentString)
		return string(data[:]), err
	}

	// fn markdown: Render the given Markdown string *value* as sanitized HTML.
	rv[`markdown`] = func(value interface{}) (template.HTML, error) {
		input := fmt.Sprintf("%v", value)
		output := blackfriday.MarkdownCommon([]byte(input[:]))
		output = bluemonday.UGCPolicy().SanitizeBytes(output)

		return template.HTML(output[:]), nil
	}

	// fn csv: Render the given *values* as a line suitable for inclusion in a common-separated
	//         values file.
	rv[`csv`] = func(header []interface{}, lines []interface{}) (string, error) {
		return delimited(',', header, lines)
	}

	// fn tsv: Render the given *values* as a line suitable for inclusion in a tab-separated
	//         values file.
	rv[`tsv`] = func(header []interface{}, lines []interface{}) (string, error) {
		return delimited('\t', header, lines)
	}

	// fn unsafe: Return an unescaped raw HTML segment for direct inclusion in the rendered
	//            template output.  This is a common antipattern that leads to all kinds of
	//            security issues from poorly-constrained implementations, so you are forced
	//            to acknowledge this by typing "unsafe".
	rv[`unsafe`] = func(value string) template.HTML {
		return template.HTML(value)
	}

	// fn sanitize: Takes a raw HTML string and santizes it, removing attributes and elements
	//              that can be used to evaluate scripts, but leaving the rest.  Useful for
	//              preparing user-generated HTML for display.
	rv[`sanitize`] = func(value string) template.HTML {
		return template.HTML(bluemonday.UGCPolicy().Sanitize(value))
	}

	// Type Handling and Conversion
	// ---------------------------------------------------------------------------------------------

	// fn isBool: Return whether the given *value* is a boolean type.
	rv[`isBool`] = stringutil.IsBoolean

	// fn isInt: Return whether the given *value* is an integer type.
	rv[`isInt`] = stringutil.IsInteger

	// fn isFloat: Return whether the given *value* is a floating-point type.
	rv[`isFloat`] = stringutil.IsFloat

	// fn isZero: Return whether the given *value* is an zero-valued variable.
	rv[`isZero`] = typeutil.IsZero

	// fn isEmpty: Return whether the given *value* is empty.
	rv[`isEmpty`] = typeutil.IsEmpty

	// fn isArray: Return whether the given *value* is an iterable array or slice.
	rv[`isArray`] = typeutil.IsArray

	// fn isMap: Return whether the given *value* is a key-value map type.
	rv[`isMap`] = func(value interface{}) bool {
		return typeutil.IsKind(value, reflect.Map)
	}

	// fn autotype: Attempt to automatically determine the type if *value* and return the converted output.
	rv[`autotype`] = stringutil.Autotype

	// fn asStr: Return the *value* as a string.
	rv[`asStr`] = stringutil.ToString

	// fn asInt: Attempt to convert the given *value* to an integer.
	rv[`asInt`] = func(value interface{}) (int64, error) {
		if v, err := stringutil.ConvertToFloat(value); err == nil {
			return int64(v), nil
		} else {
			return 0, err
		}
	}

	// fn asFloat: Attempt to convert the given *value* to a floating-point number.
	rv[`asFloat`] = stringutil.ConvertToFloat

	// fn asBool: Attempt to convert the given *value* to a boolean value.
	rv[`asBool`] = stringutil.ConvertToBool

	// fn asTime: Attempt to parse the given *value* as a date/time value.
	rv[`asTime`] = stringutil.ConvertToTime

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

	// Time and Date Formatting
	// ---------------------------------------------------------------------------------------------

	// fn time: Return the given Time formatted using *format*.  See [Time Formats](#time-formats) for
	//          acceptable formats.
	rv[`time`] = tmFmt

	// fn now: Return the current time formatted using *format*.  See [Time Formats](#time-formats) for
	//          acceptable formats.
	rv[`now`] = func(format ...string) (string, error) {
		return tmFmt(time.Now(), format...)
	}

	// fn ago: Return a Time subtracted by the given *duration*.
	rv[`ago`] = func(durationString string, fromTime ...time.Time) (time.Time, error) {
		from := time.Now()

		if len(fromTime) > 0 {
			from = fromTime[0]
		}

		if duration, err := time.ParseDuration(durationString); err == nil {
			return from.Add(-1 * duration), nil
		} else {
			return time.Time{}, err
		}
	}

	// fn since: Return the amount of time that has elapsed since *time*, optionally rounded
	//           to the nearest *interval*.
	rv[`since`] = func(at interface{}, interval ...string) (time.Duration, error) {
		if tm, err := stringutil.ConvertToTime(at); err == nil {
			since := time.Since(tm)

			if len(interval) > 0 {
				switch strings.ToLower(interval[0]) {
				case `s`, `sec`, `second`:
					since = since.Round(time.Second)
				case `m`, `min`, `minute`:
					since = since.Round(time.Minute)
				case `h`, `hr`, `hour`:
					since = since.Round(time.Hour)
				}
			}

			return since, nil
		} else {
			return 0, err
		}
	}

	// fn duration: Convert the given *value* from a duration of *unit* into the given time *format*.
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

	// Random Numbers and Encoding
	// ---------------------------------------------------------------------------------------------

	// fn random: Return a random array of *n* bytes. The random source used is suitable for
	//            cryptographic purposes.
	rv[`random`] = func(count int) ([]byte, error) {
		output := make([]byte, count)
		if _, err := rand.Read(output); err == nil {
			return output, nil
		} else {
			return nil, err
		}
	}

	// fn uuid: Generate a new Version 4 UUID.
	rv[`uuid`] = func() (string, error) {
		if u, err := uuid.NewV4(); err == nil {
			return u.String(), nil
		} else {
			return ``, err
		}
	}

	// fn uuidRaw: Generate the raw bytes of a new Version 4 UUID.
	rv[`uuidRaw`] = func() ([]byte, error) {
		if u, err := uuid.NewV4(); err == nil {
			return u.Bytes(), nil
		} else {
			return nil, err
		}
	}

	// fn base32: Encode the *input* bytes with the Base32 encoding scheme.
	rv[`base32`] = func(input []byte) string {
		return Base32Alphabet.EncodeToString(input)
	}

	// fn base58: Encode the *input* bytes with the Base58 (Bitcoin alphabet) encoding scheme.
	rv[`base58`] = func(input []byte) string {
		return base58.Encode(input)
	}

	// fn base64: Encode the *input* bytes with the Base64 encoding scheme.  Optionally specify
	//            the encoding mode: one of "padded", "url", "url-padded", or empty (unpadded, default).
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

	// fn murmur3: hash the *input* data using the Murmur 3 algorithm.
	rv[`murmur3`] = func(input interface{}) (uint64, error) {
		if v, err := stringutil.ToString(input); err == nil {
			return murmur3.Sum64([]byte(v)), nil
		} else {
			return 0, err
		}
	}

	// TODO:
	// urlencode/urldecode
	// rv[`md5`] =
	// rv[`sha1`] =
	// rv[`sha256`] =
	// rv[`sha384`] =
	// rv[`sha512`] =

	// Numeric/Math Functions
	// ---------------------------------------------------------------------------------------------
	rv[`calc`] = calcFn

	// fn add: Return the sum of all of the given *values*.
	rv[`add`] = func(values ...interface{}) float64 {
		out, _ := calcFn(`+`, values...)
		return out
	}

	// fn subtract: Sequentially subtract all of the given *values*.
	rv[`subtract`] = func(values ...interface{}) float64 {
		out, _ := calcFn(`-`, values...)
		return out
	}

	// fn multiply: Return the product of all of the given *values*.
	rv[`multiply`] = func(values ...interface{}) float64 {
		out, _ := calcFn(`*`, values...)
		return out
	}

	// fn divide: Sequentially divide all of the given *values*.
	rv[`divide`] = func(values ...interface{}) (float64, error) {
		return calcFn(`/`, values...)
	}

	// fn mod: Return the modulus of all of the given *values*.
	rv[`mod`] = func(values ...interface{}) (float64, error) {
		return calcFn(`%`, values...)
	}

	// fn pow: Sequentially exponentiate of all of the given *values*.
	rv[`pow`] = func(values ...interface{}) (float64, error) {
		return calcFn(`^`, values...)
	}

	// fn sequence: Return an array of integers representing a sequence from [0, *n*).
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

	// Numeric Aggregation Functions
	// ---------------------------------------------------------------------------------------------
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

	// Simplified Comparators
	// ---------------------------------------------------------------------------------------------

	// fn eqx: A relaxed-type version of the **eq** builtin function.
	rv[`eqx`] = stringutil.RelaxedEqual

	// fn nex: A relaxed-type version of the **ne** builtin function.
	rv[`nex`] = func(first interface{}, second interface{}) (bool, error) {
		eq, err := stringutil.RelaxedEqual(first, second)
		return !eq, err
	}

	// Set Processing
	// ---------------------------------------------------------------------------------------------

	// fn filter: Return the given *input* array with only elements where *expression* evaluates to
	//            a truthy value.
	rv[`filter`] = func(input interface{}, expr string) ([]interface{}, error) {
		out := make([]interface{}, 0)

		for i, value := range sliceutil.Sliceify(input) {
			tmpl := NewTemplate(`inline`, TextEngine)
			tmpl.Funcs(rv)

			if !strings.HasPrefix(expr, `{{`) {
				expr = `{{` + expr
			}

			if !strings.HasSuffix(expr, `}}`) {
				expr = expr + `}}`
			}

			if err := tmpl.Parse(expr); err == nil {
				output := bytes.NewBuffer(nil)

				if err := tmpl.Render(output, value, ``); err == nil {
					evalValue := stringutil.Autotype(output.String())

					if !typeutil.IsZero(evalValue) {
						out = append(out, value)
					}
				} else {
					return nil, fmt.Errorf("item %d: %v", i, err)
				}
			} else {
				return nil, fmt.Errorf("failed to parse template: %v", err)
			}
		}

		return out, nil
	}

	// fn filterByKey: Return a subset of the elements in the *input* array whose map values
	//                 contain the *key*, optionally matching *expression*.
	rv[`filterByKey`] = func(input interface{}, key string, exprs ...interface{}) ([]interface{}, error) {
		return filterByKey(rv, input, key, exprs...)
	}

	// fn firstByKey: Return the first elements in the *input* array whose map values
	//                 contain the *key*, optionally matching *expression*.
	rv[`firstByKey`] = func(input interface{}, key string, exprs ...interface{}) (interface{}, error) {
		if v, err := filterByKey(rv, input, key, exprs...); err == nil {
			return sliceutil.First(v), nil
		} else {
			return nil, err
		}
	}

	// fn pluck: Given an *input* array of maps, retrieve the values of *key* from all elements.
	rv[`pluck`] = func(input interface{}, key string) []interface{} {
		return maputil.Pluck(input, strings.Split(key, `.`))
	}

	// fn get: Get a key from a map.
	rv[`get`] = func(input interface{}, key string, fallback ...interface{}) interface{} {
		var fb interface{}

		if len(fallback) > 0 {
			fb = fallback[0]
		}

		return maputil.DeepGet(input, strings.Split(key, `.`), fb)
	}

	// fn findkey: Recursively scans the given *input* array or map and returns all values of the given *key*.
	rv[`findkey`] = func(input interface{}, key string) ([]interface{}, error) {
		values := make([]interface{}, 0)

		if err := maputil.Walk(input, func(value interface{}, path []string, isLeaf bool) error {
			if isLeaf && path[len(path)-1] == key {
				values = append(values, value)
			}

			return nil
		}); err != nil {
			return nil, err
		}

		return values, nil
	}

	// fn has: Return whether *want* is an element of the given *input* array.
	rv[`has`] = func(want interface{}, input interface{}) bool {
		for _, have := range sliceutil.Sliceify(input) {
			if eq, err := stringutil.RelaxedEqual(have, want); err == nil && eq == true {
				return true
			}
		}

		return false
	}

	// fn any: Return whether *input* array contains any of the the elements *wanted*.
	rv[`any`] = func(input interface{}, wants ...interface{}) bool {
		for _, have := range sliceutil.Sliceify(input) {
			for _, want := range wants {
				if eq, err := stringutil.RelaxedEqual(have, want); err == nil && eq == true {
					return true
				}
			}
		}

		return false
	}

	// fn indexOf: Iterate through the *input* array and return the index of *value*, or -1 if not present.
	rv[`indexOf`] = func(slice interface{}, value interface{}) (index int) {
		index = -1

		if typeutil.IsArray(slice) {
			sliceutil.Each(slice, func(i int, v interface{}) error {
				if eq, err := stringutil.RelaxedEqual(v, value); err == nil && eq == true {
					index = i
					return sliceutil.Stop
				} else {
					return nil
				}
			})
		}

		return
	}

	// fn uniq: Return an array of unique values from the given *input* array.
	rv[`uniq`] = func(slice interface{}) []interface{} {
		return sliceutil.Unique(slice)
	}

	// fn flatten: Return an array of values with all nested subarrays merged into a single level.
	rv[`flatten`] = func(slice interface{}) []interface{} {
		return sliceutil.Flatten(slice)
	}

	// fn compact: Return an copy of given *input* array with all zero-valued elements removed.
	rv[`compact`] = func(slice interface{}) []interface{} {
		return sliceutil.Compact(slice)
	}

	// fn first: Return the first value from the given *input* array.
	rv[`first`] = func(slice interface{}) (out interface{}, err error) {
		err = sliceutil.Each(slice, func(i int, value interface{}) error {
			out = value
			return sliceutil.Stop
		})

		return
	}

	// fn last: Return the last value from the given *input* array.
	rv[`last`] = func(slice interface{}) (out interface{}, err error) {
		err = sliceutil.Each(slice, func(i int, value interface{}) error {
			out = value
			return nil
		})

		return
	}

	// fn count: A type-relaxed version of **len**.
	rv[`count`] = func(in interface{}) int {
		return sliceutil.Len(in)
	}

	// fn sort: Return the *input* array sorted in lexical ascending order.
	rv[`sort`] = func(input interface{}, keys ...string) []interface{} {
		return sorter(input, false, keys...)
	}

	// fn reverse: Return the *input* array sorted in lexical descending order.
	rv[`reverse`] = func(input interface{}, keys ...string) []interface{} {
		return sorter(input, true, keys...)
	}

	// fn isort: Return the *input* array sorted in lexical ascending order (case insensitive).
	rv[`isort`] = func(input interface{}, keys ...string) []interface{} {
		return sorter(sliceutil.MapString(input, func(_ int, v string) string {
			return strings.ToLower(v)
		}), true, keys...)
	}

	// fn ireverse: Return the *input* array sorted in lexical descending order (case insensitive).
	rv[`ireverse`] = func(input interface{}, keys ...string) []interface{} {
		return sorter(sliceutil.MapString(input, func(_ int, v string) string {
			return strings.ToLower(v)
		}), true, keys...)
	}

	// fn mostcommon: Return element in the *input* array that appears the most frequently.
	rv[`mostcommon`] = func(slice interface{}) (interface{}, error) {
		return commonses(slice, `most`)
	}

	// fn leastcommon: Return element in the *input* array that appears the least frequently.
	rv[`leastcommon`] = func(slice interface{}) (interface{}, error) {
		return commonses(slice, `least`)
	}

	// fn stringify: Return the given *input* array with all values converted to strings.
	rv[`stringify`] = func(slice interface{}) []string {
		return sliceutil.Stringify(slice)
	}

	return rv
}
