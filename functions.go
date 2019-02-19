package diecast

import (
	"bytes"
	"encoding/base32"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"math"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	humanize "github.com/dustin/go-humanize"
	"github.com/ghetzel/go-stockutil/fileutil"
	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/typeutil"
	"github.com/kelvins/sunrisesunset"
	"github.com/montanaflynn/stats"
	"github.com/russross/blackfriday/v2"
	"golang.org/x/net/html"
)

var Base32Alphabet = base32.NewEncoding(`abcdefghijklmnopqrstuvwxyz234567`)

type fileInfo struct {
	Parent    string
	Directory bool
	os.FileInfo
}

func (self *fileInfo) MarshalJSON() ([]byte, error) {
	full := path.Join(self.Parent, self.Name())

	data := map[string]interface{}{
		`name`:          self.Name(),
		`path`:          full,
		`size`:          self.Size(),
		`last_modified`: self.ModTime(),
		`directory`:     self.IsDir(),
	}

	if !self.IsDir() {
		data[`mimetype`] = fileutil.GetMimeType(full)
	}

	return json.Marshal(data)
}

func (self *fileInfo) String() string {
	return path.Join(self.Parent, self.Name())
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

func GetFunctions() (funcGroups, FuncMap) {
	funcs := make(FuncMap)
	groups := make(funcGroups, 0)

	// String Processing
	groups = append(groups, loadStandardFunctionsString(funcs))

	// File Pathname Handling
	groups = append(groups, loadStandardFunctionsPath(funcs))

	// Encoding / Decoding
	groups = append(groups, loadStandardFunctionsCodecs(funcs))

	// Type Handling and Conversion
	groups = append(groups, loadStandardFunctionsTypes(funcs))

	// Time and Date Formatting
	groups = append(groups, loadStandardFunctionsTime(funcs))

	// Random Numbers and Encoding
	groups = append(groups, loadStandardFunctionsCryptoRand(funcs))

	// Numeric/Math Functions
	groups = append(groups, loadStandardFunctionsMath(funcs))

	// Collections
	groups = append(groups, loadStandardFunctionsCollections(funcs))

	// HTML processing
	groups = append(groups, loadStandardFunctionsHtmlProcessing(funcs))

	// Colors
	groups = append(groups, loadStandardFunctionsColor(funcs))

	// Unit Conversions
	groups = append(groups, loadStandardFunctionsConvert(funcs))

	// Template Introspection functions
	groups = append(groups, loadStandardFunctionsIntrospection(funcs))

	// Comparators
	groups = append(groups, loadStandardFunctionsComparisons(funcs))

	// Documentation for runtime functions
	groups = append(groups, loadRuntimeFunctionsVariables())
	groups = append(groups, loadRuntimeFunctionsRequest())

	groups.PopulateFuncMap(funcs)

	return groups, funcs
}

func GetStandardFunctions() FuncMap {
	_, funcs := GetFunctions()
	return funcs
}

type statsTplFunc func(in interface{}) (float64, error) // {}

func delimited(comma rune, header []interface{}, lines []interface{}) (string, error) {
	output := bytes.NewBufferString(``)
	csvwriter := csv.NewWriter(output)
	csvwriter.Comma = comma
	csvwriter.UseCRLF = true
	input := make([][]string, 0)

	columnNames := sliceutil.Stringify(header)
	input = append(input, columnNames)

	for _, line := range lines {
		lineslice := sliceutil.Sliceify(line)

		for i, value := range lineslice {
			if typeutil.IsArray(value) && len(sliceutil.Compact(sliceutil.Sliceify(value))) == 0 {
				if i+1 < len(lineslice) {
					lineslice = append(lineslice[:i], lineslice[i+1:]...)
				} else {
					lineslice = lineslice[:i]
				}
			} else if typeutil.IsMap(value) {
				m := maputil.M(value)

				for j, col := range columnNames {
					if j < len(lineslice) {
						lineslice[j] = m.Get(col)
					}
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
			case `rfc1123`:
				tmFormat = time.RFC1123
			case `rfc1123z`:
				tmFormat = time.RFC1123Z
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
			case `ansi`, `ansic`:
				tmFormat = time.ANSIC
			case `unixdate`:
				tmFormat = time.UnixDate
			case `stamp`:
				tmFormat = time.Stamp
			case `stamp-ms`:
				tmFormat = time.StampMilli
			case `stamp-us`:
				tmFormat = time.StampMicro
			case `stamp-ns`:
				tmFormat = time.StampNano
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

	for i, mapitem := range sliceutil.Sliceify(input) {
		submap := maputil.M(mapitem)

		if item := submap.Get(key); !item.IsNil() {
			if stringutil.IsSurroundedBy(exprStr, `{{`, `}}`) {
				tmpl := NewTemplate(`inline`, TextEngine)
				tmpl.Funcs(funcs)

				if err := tmpl.Parse(exprStr); err == nil {
					output := bytes.NewBuffer(nil)

					if err := tmpl.Render(output, item.Value, ``); err == nil {
						if evalValue := stringutil.Autotype(output.String()); !typeutil.IsZero(evalValue) {
							out = append(out, mapitem)
						}
					} else {
						return nil, fmt.Errorf("item %d: %v", i, err)
					}
				} else {
					return nil, fmt.Errorf("failed to parse template: %v", err)
				}
			} else if typeutil.IsArray(expr) {
				// if we were given an array, then matching ANY item in the array yields true
				for _, want := range sliceutil.Sliceify(expr) {
					if ok, err := stringutil.RelaxedEqual(item, want); err == nil && ok {
						out = append(out, mapitem)
						break
					}
				}

			} else if ok, err := stringutil.RelaxedEqual(item, expr); err == nil && ok {
				out = append(out, mapitem)
			}
		}
	}

	return out, nil
}

func uniqByKey(funcs FuncMap, input interface{}, key string, saveLast bool, exprs ...interface{}) ([]interface{}, error) {
	out := make([]interface{}, 0)
	expr := sliceutil.First(exprs)
	exprStr := fmt.Sprintf("%v", expr)
	valuesEncountered := make(map[string]int)

	for i, submap := range sliceutil.Sliceify(input) {
		if typeutil.IsMap(submap) {
			if item := maputil.DeepGet(submap, strings.Split(key, `.`)); item != nil {
				var valkey string

				if stringutil.IsSurroundedBy(exprStr, `{{`, `}}`) {
					tmpl := NewTemplate(`inline`, TextEngine)
					tmpl.Funcs(funcs)

					if err := tmpl.Parse(exprStr); err == nil {
						output := bytes.NewBuffer(nil)

						if err := tmpl.Render(output, item, ``); err == nil {
							valkey = output.String()
						} else {
							return nil, fmt.Errorf("item %d: %v", i, err)
						}
					} else {
						return nil, fmt.Errorf("failed to parse template: %v", err)
					}
				} else {
					valkey = fmt.Sprintf("%v", item)
				}

				// if we're saving the last value, then always overwrite; otherwise, only
				// mark this item for inclusion in the output if nothing else has been in this
				// spot before
				if _, ok := valuesEncountered[valkey]; saveLast || !ok {
					valuesEncountered[valkey] = i
				}
			}
		}
	}

	// put only the unique values into the output
	for i, submap := range sliceutil.Sliceify(input) {
		for _, vi := range valuesEncountered {
			if i == vi {
				out = append(out, submap)
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

func getSunriseSunset(latitude float64, longitude float64, atTime ...interface{}) (time.Time, time.Time, error) {
	var at time.Time

	if len(atTime) > 0 {
		if tm, err := stringutil.ConvertToTime(atTime[0]); err == nil {
			at = tm
		} else {
			return time.Time{}, time.Time{}, err
		}
	} else {
		at = time.Now()
	}

	_, offset := at.Zone()

	p := sunrisesunset.Parameters{
		Latitude:  latitude,
		Longitude: longitude,
		UtcOffset: (float64(offset) / 60.0 / 60.0),
		Date:      at,
	}

	if sunrise, sunset, err := p.GetSunriseSunset(); err == nil {
		sunrise = time.Date(at.Year(), at.Month(), at.Day(), sunrise.Hour(), sunrise.Minute(), sunrise.Second(), 0, at.Location())
		sunset = time.Date(at.Year(), at.Month(), at.Day(), sunset.Hour(), sunset.Minute(), sunset.Second(), 0, at.Location())

		return sunrise, sunset, nil
	} else {
		return time.Time{}, time.Time{}, err
	}
}

func timeCmp(before bool, first interface{}, secondI ...interface{}) (bool, error) {
	var second interface{}

	if len(secondI) == 0 {
		second = first
		first = time.Now()
	} else {
		second = secondI[0]
	}

	if firstT, err := stringutil.ConvertToTime(first); err == nil {
		if secondT, err := stringutil.ConvertToTime(second); err == nil {
			if before {
				return firstT.Before(secondT), nil
			} else {
				return firstT.After(secondT), nil
			}
		} else {
			return false, err
		}

	} else {
		return false, err
	}
}

func cmp(op string, first interface{}, second interface{}) (bool, error) {
	fStr, ok1 := first.(string)
	sStr, ok2 := second.(string)

	if ok1 && ok2 {
		switch op {
		case `gt`:
			return fStr > sStr, nil
		case `ge`:
			return fStr >= sStr, nil
		case `lt`:
			return fStr < sStr, nil
		case `le`:
			return fStr <= sStr, nil
		default:
			return false, fmt.Errorf("Invalid operator %q", op)
		}
	} else {
		fVal := typeutil.Float(first)
		sVal := typeutil.Float(second)

		switch op {
		case `gt`:
			return fVal > sVal, nil
		case `ge`:
			return fVal >= sVal, nil
		case `lt`:
			return fVal < sVal, nil
		case `le`:
			return fVal <= sVal, nil
		default:
			return false, fmt.Errorf("Invalid operator %q", op)
		}
	}
}

func toMarkdownExt(extensions ...string) blackfriday.Extensions {
	var ext blackfriday.Extensions

	for _, x := range extensions {
		switch stringutil.Hyphenate(x) {
		case `no-intra-emphasis`:
			ext |= blackfriday.NoIntraEmphasis
		case `tables`:
			ext |= blackfriday.Tables
		case `fenced-code`:
			ext |= blackfriday.FencedCode
		case `autolink`:
			ext |= blackfriday.Autolink
		case `strikethrough`:
			ext |= blackfriday.Strikethrough
		case `lax-html-blocks`:
			ext |= blackfriday.LaxHTMLBlocks
		case `space-headings`:
			ext |= blackfriday.SpaceHeadings
		case `hard-line-break`:
			ext |= blackfriday.HardLineBreak
		case `tab-size-eight`:
			ext |= blackfriday.TabSizeEight
		case `footnotes`:
			ext |= blackfriday.Footnotes
		case `no-empty-line-before-block`:
			ext |= blackfriday.NoEmptyLineBeforeBlock
		case `heading-ids`:
			ext |= blackfriday.HeadingIDs
		case `titleblock`:
			ext |= blackfriday.Titleblock
		case `auto-heading-ids`:
			ext |= blackfriday.AutoHeadingIDs
		case `backslash-line-break`:
			ext |= blackfriday.BackslashLineBreak
		case `definition-lists`:
			ext |= blackfriday.DefinitionLists
		case `common`:
			ext |= blackfriday.CommonExtensions
		}
	}

	return ext
}

func htmldoc(docI interface{}) (*goquery.Document, error) {
	if d, ok := docI.(*goquery.Document); ok {
		return d, nil
	} else if d, ok := docI.(string); ok {
		return goquery.NewDocumentFromReader(bytes.NewBufferString(d))
	} else if d, ok := docI.(template.HTML); ok {
		return goquery.NewDocumentFromReader(bytes.NewBufferString(string(d)))
	} else {
		return nil, fmt.Errorf("Expected a HTML document string or object, got: %T", docI)
	}
}

func htmlModify(docI interface{}, selector string, action string, k string, v interface{}, extra ...interface{}) (template.HTML, error) {
	if doc, err := htmldoc(docI); err == nil {
		switch action {
		case `remove`:
			doc.Find(selector).Remove()
		case `add-class`:
			doc.Find(selector).AddClass(sliceutil.Stringify(sliceutil.Flatten(v))...)
		case `remove-class`:
			doc.Find(selector).RemoveClass(sliceutil.Stringify(sliceutil.Flatten(v))...)
		case `set-attr`:
			doc.Find(selector).SetAttr(k, typeutil.String(v))
		case `find-replace-attr`:
			if len(extra) > 0 {
				if rxFind, err := regexp.Compile(typeutil.String(extra[0])); err == nil {
					doc.Find(selector).Each(func(i int, match *goquery.Selection) {
						if current, ok := match.Attr(k); ok {
							match.SetAttr(k, rxFind.ReplaceAllString(current, typeutil.String(v)))
						}
					})
				} else {
					return ``, fmt.Errorf("invalid find expression: %v", err)
				}
			} else {
				return ``, fmt.Errorf("no find expression specified")
			}
		case `find-replace-text`:
			if len(extra) > 0 {
				if rxFind, err := regexp.Compile(typeutil.String(extra[0])); err == nil {
					doc.Find(selector).Each(func(i int, match *goquery.Selection) {
						for _, node := range match.Nodes {
							// recursively walk the subtree from this node and apply the
							// find/replace to all text therein
							walkNodeTree(node, func(n *html.Node) bool {
								switch n.Type {
								case html.TextNode:
									n.Data = rxFind.ReplaceAllString(n.Data, typeutil.String(v))
								}

								return true
							})
						}
					})
				} else {
					return ``, fmt.Errorf("invalid find expression: %v", err)
				}
			} else {
				return ``, fmt.Errorf("no find expression specified")
			}
		default:
			return ``, fmt.Errorf("unknown HTML action %q", action)
		}

		doc.End()
		output, err := doc.Html()

		return template.HTML(output), err
	} else {
		return ``, err
	}
}

// recursively walk a subtree starting from a given node, calling fn for each node
// (including the entry point).
func walkNodeTree(node *html.Node, fn func(child *html.Node) bool) {
	if !fn(node) {
		return
	}

	switch node.Type {
	case html.ElementNode:
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			if !fn(child) {
				return
			}
		}
	}
}

func toBytes(input interface{}) []byte {
	var in []byte

	if v, ok := input.([]byte); ok {
		in = v
	} else {
		in = []byte(typeutil.String(input))
	}

	return in
}
