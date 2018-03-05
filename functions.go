package diecast

import (
	"bytes"
	"encoding/base32"
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	humanize "github.com/dustin/go-humanize"
	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/typeutil"
	"github.com/montanaflynn/stats"
	"golang.org/x/net/html"
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
	loadStandardFunctionsString(rv)

	// File Pathname Handling
	loadStandardFunctionsPath(rv)

	// Encoding / Decoding
	loadStandardFunctionsCodecs(rv)

	// Type Handling and Conversion
	loadStandardFunctionsTypes(rv)

	// Time and Date Formatting
	loadStandardFunctionsTime(rv)

	// Random Numbers and Encoding
	loadStandardFunctionsCryptoRand(rv)

	// Numeric/Math Functions
	loadStandardFunctionsMath(rv)

	// Collections
	loadStandardFunctionsCollections(rv)

	// Web Scraping
	loadStandardFunctionsWebScraping(rv)

	// Colors
	loadStandardFunctionsColor(rv)

	// Miscellaneous
	loadStandardFunctionsMisc(rv)

	return rv
}

type statsTplFunc func(in interface{}) (float64, error) // {}

func delimited(comma rune, header []interface{}, lines []interface{}) (string, error) {
	output := bytes.NewBufferString(``)
	csvwriter := csv.NewWriter(output)
	csvwriter.Comma = comma
	csvwriter.UseCRLF = true
	input := make([][]string, 0)

	input = append(input, sliceutil.Stringify(header))

	for _, line := range lines {
		lineslice := sliceutil.Sliceify(line)

		for i, value := range lineslice {
			if typeutil.IsArray(value) && len(sliceutil.Compact(sliceutil.Sliceify(value))) == 0 {
				if i+1 < len(lineslice) {
					lineslice = append(lineslice[:i], lineslice[i+1:]...)
				} else {
					lineslice = lineslice[:i]
				}
			}
		}

		input = append(input, sliceutil.Stringify(
			sliceutil.Flatten(lineslice),
		))
	}

	if err := csvwriter.WriteAll(input); err != nil {
		return ``, err
	}

	return output.String(), nil
}

func tmFmt(value interface{}, format ...string) (string, error) {
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
			case `epoch-ns`:
				return fmt.Sprintf("%d", int64(v.UnixNano())), nil
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

		var vStr string

		switch tmFormat {
		case `human`:
			vStr = humanize.Time(v)
		default:
			vStr = v.Format(tmFormat)
		}

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

func calcFn(op string, values ...interface{}) (float64, error) {
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

func filterByKey(funcs FuncMap, input interface{}, key string, exprs ...interface{}) ([]interface{}, error) {
	out := make([]interface{}, 0)
	expr := sliceutil.First(exprs)
	exprStr := fmt.Sprintf("%v", expr)

	for i, submap := range sliceutil.Sliceify(input) {
		if typeutil.IsMap(submap) {
			if item := maputil.DeepGet(submap, strings.Split(key, `.`)); item != nil {

				if stringutil.IsSurroundedBy(exprStr, `{{`, `}}`) {
					tmpl := NewTemplate(`inline`, TextEngine)
					tmpl.Funcs(funcs)

					if err := tmpl.Parse(exprStr); err == nil {
						output := bytes.NewBuffer(nil)

						if err := tmpl.Render(output, item, ``); err == nil {
							evalValue := stringutil.Autotype(output.String())

							if !typeutil.IsZero(evalValue) {
								out = append(out, submap)
							}
						} else {
							return nil, fmt.Errorf("item %d: %v", i, err)
						}
					} else {
						return nil, fmt.Errorf("failed to parse template: %v", err)
					}
				} else if ok, err := stringutil.RelaxedEqual(item, expr); err == nil && ok {
					out = append(out, submap)
				}
			}
		}
	}

	return out, nil
}

func sorter(input interface{}, reverse bool, keys ...string) []interface{} {
	out := sliceutil.Sliceify(input)

	sort.Slice(out, func(i, j int) bool {
		var iVal, jVal string

		if len(keys) > 0 {
			iVal = maputil.DeepGetString(out[i], strings.Split(keys[0], `.`))
			jVal = maputil.DeepGetString(out[j], strings.Split(keys[0], `.`))
		} else {
			iVal, _ = stringutil.ToString(out[i])
			jVal, _ = stringutil.ToString(out[j])
		}

		if reverse {
			return iVal > jVal
		} else {
			return iVal < jVal
		}
	})

	return out
}

func commonses(slice interface{}, cmp string) (interface{}, error) {
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

func htmlNodeToMap(node *html.Node) map[string]interface{} {
	output := make(map[string]interface{})

	if node != nil && node.Type == html.ElementNode {
		text := ``
		children := make([]map[string]interface{}, 0)
		attrs := make(map[string]interface{})

		for child := node.FirstChild; child != nil; child = child.NextSibling {
			switch child.Type {
			case html.TextNode:
				text += child.Data
			case html.ElementNode:
				if child != node {
					if childData := htmlNodeToMap(child); len(childData) > 0 {
						children = append(children, childData)
					}
				}
			}
		}

		text = strings.TrimSpace(text)

		for _, attr := range node.Attr {
			attrs[attr.Key] = stringutil.Autotype(attr.Val)
		}

		if len(attrs) > 0 {
			output[`attributes`] = attrs
		}

		if text != `` {
			output[`text`] = text
		}

		if len(children) > 0 {
			output[`children`] = children
		}

		// only if the node has anything useful at all in it...
		if len(output) > 0 {
			output[`name`] = node.DataAtom.String()
		}
	}

	return output
}
