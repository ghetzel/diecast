package diecast

import (
	"bytes"
	"fmt"
	"math/rand"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/typeutil"
)

var errorInterface = reflect.TypeOf((*error)(nil)).Elem()

func loadStandardFunctionsCollections(funcs FuncMap, server *Server) funcGroup {
	var group = funcGroup{
		Name: `Arrays and Objects`,
		Description: `For converting, modifying, and filtering arrays, objects, and arrays of ` +
			`objects. These functions are especially useful when working with data returned from Bindings.`,
		Functions: []funcDef{
			{
				Name: `append`,
				Summary: `Append one or more values to the given array.  If the array given is not in fact an array, ` +
					`it will be converted into one, with the exception of null values, which will create an empty array.`,
				Arguments: []funcArg{
					{
						Name:        `array`,
						Type:        `array`,
						Description: `The array to append items to.`,
					}, {
						Name:        `values`,
						Type:        `any`,
						Variadic:    true,
						Description: `One or more items to append to the given array.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `append ["a", "b"] "c" "d"`,
						Return: []string{`a`, `b`, `c`, `d`},
					},
				},
				Function: func(array interface{}, values ...interface{}) ([]interface{}, error) {
					var out = make([]interface{}, 0)

					if array != nil && !typeutil.IsArray(array) {
						out = sliceutil.Sliceify(array)
					}

					out = append(out, values...)

					return out, nil
				},
			}, {
				Name: `page`,
				Summary: `Returns an integer representing an offset used for accessing paginated values when ` +
					`given a page number and number of results per page.`,
				Arguments: []funcArg{
					{
						Name:        `pagenum`,
						Type:        `integer`,
						Description: `The page number to calculate the offset of.`,
					}, {
						Name:        `perpage`,
						Type:        `integer`,
						Description: `The maximum number of results that can appear on a single page.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `page 1 25`,
						Return: 0,
					}, {
						Code:   `page 2 25`,
						Return: 25,
					}, {
						Code:   `page 3 25`,
						Return: 50,
					}, {
						Code:   `page 2 10`,
						Return: 10,
					},
				},
				Function: func(pagenum interface{}, perpage interface{}) int {
					var factor = typeutil.V(pagenum).Int() - 1
					var per = typeutil.V(perpage).Int()

					if factor >= 0 {
						if per > 0 {
							return int(factor * per)
						}
					}

					return 0
				},
			}, {
				Name:    `reverse`,
				Summary: `Return the given array in reverse order.`,
				Arguments: []funcArg{
					{
						Name:        `array`,
						Type:        `array`,
						Description: `The array to reverse.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `reverse [1,2,3]`,
						Return: []int{3, 2, 1},
					},
				},
				Function: func(input interface{}) []interface{} {
					var array = sliceutil.Sliceify(input)
					var output = make([]interface{}, len(array))

					for i := 0; i < len(array); i++ {
						output[len(array)-1-i] = array[i]
					}

					return output
				},
			}, {
				Name:    `filter`,
				Summary: `Return the given array with only elements where expression evaluates to a truthy value.`,
				Arguments: []funcArg{
					{
						Name:        `array`,
						Type:        `array`,
						Description: `The array to operate on.`,
					}, {
						Name: `expression`,
						Type: `string`,
						Description: `An "{{ expression }}" that will be called on each element.  Only if the ` +
							`expression does *not* yield a zero value (0, false, "", null) will the element be included ` +
							`in the resulting array.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `filter [1, 2, 3, 4, 5] "{{ isOdd . }}"`,
						Return: []int{1, 3, 5},
					}, {
						Code: `filter [{"active": true, "a": 1}, {"b": 2}, {"active": true, "c": 3}] "{{ .active }}"`,
						Return: []map[string]interface{}{
							{"active": true, "a": 1},
							{"active": true, "c": 3},
						},
					},
				},
				Function: func(input interface{}, expr string) ([]interface{}, error) {
					var out = make([]interface{}, 0)

					for i, value := range sliceutil.Sliceify(input) {
						var tmpl = NewTemplate(`inline`, TextEngine)
						tmpl.Funcs(funcs)

						if !strings.HasPrefix(expr, `{{`) {
							expr = `{{` + expr
						}

						if !strings.HasSuffix(expr, `}}`) {
							expr = expr + `}}`
						}

						if err := tmpl.ParseString(expr); err == nil {
							var output = bytes.NewBuffer(nil)

							if err := tmpl.Render(output, value, ``); err == nil {
								var evalValue = stringutil.Autotype(output.String())

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
				},
			}, {
				Name: `filterLines`,
				Summary: `Return a subset of the lines in the given string or array of strings that ` +
					`match the supplied regular expression.  Optionally, the match may be negated, which ` +
					`return all lines that do NOT match the regular expression.`,
				Arguments: []funcArg{
					{
						Name:        `arrayOrString`,
						Type:        `array, string`,
						Description: `The string or array of strings to scan.`,
					}, {
						Name:        `regexp`,
						Type:        `string`,
						Description: `A regular expression used to filter the given input.`,
					}, {
						Name:        `negate`,
						Type:        `boolean`,
						Description: `If true, only values that do not match regexp will be returned.`,
						Optional:    true,
					},
				},
				Examples: []funcExample{
					{
						Code: `filterLines "# Hello\n# Author: Me\necho hello\nexit 1" "^#"`,
						Return: []string{
							`# Hello`,
							`# Author: Me`,
						},
					}, {
						Code: `filterLines "# Hello\n# Author: Me\necho hello\nexit 1" "^#" true`,
						Return: []string{
							`echo hello`,
							`exit 1`,
						},
					},
				},
				Function: func(in interface{}, expr string, negate ...bool) ([]string, error) {
					if rx, err := regexp.Compile(expr); err == nil {
						var lines []string
						var doNegate bool = (len(negate) > 0 && negate[0])

						if typeutil.IsArray(in) {
							lines = sliceutil.Stringify(in)
						} else {
							lines = strings.Split(typeutil.String(in), "\n")
						}

						var out []string

						for _, line := range lines {
							if doNegate && rx.MatchString(line) {
								continue
							} else if !doNegate && !rx.MatchString(line) {
								continue
							}

							out = append(out, line)
						}

						return out, nil
					} else {
						return nil, err
					}
				},
			}, {
				Name: `filterByKey`,
				Summary: `Return a subset of the elements in the given array whose values are objects ` +
					`that contain the given key.  Optionally, the values at the key for each object in ` +
					`the array can be passed to a template expression.  If that expression produces a ` +
					`truthy value, the object will be included in the output.  Otherwise it will not.`,
				Arguments: []funcArg{
					{
						Name:        `array`,
						Type:        `array`,
						Description: `The array of objects to filter.`,
					}, {
						Name:        `key`,
						Type:        `string`,
						Description: `The name of the key on each object in the given array to check the value of.`,
					}, {
						Name: `expression`,
						Type: `string`,
						Description: `The "{{ expression }}" to apply to the value at key from each object.  ` +
							`Uses the same expression rules as [filter](#fn-filter)`,
					},
				},
				Examples: []funcExample{
					{
						Code: `filterByKey [{"id": "a", "value": 1}, {"id": "b", "value": 1}, {"id": "c", "value": 2}] 1`,
						Return: []map[string]interface{}{
							{"id": "a", "value": 1},
							{"id": "b", "value": 1},
						},
					},
				},
				Function: func(input interface{}, key string, exprs ...interface{}) ([]interface{}, error) {
					return filterByKey(funcs, input, key, exprs...)
				},
			}, {
				Name: `firstByKey`,
				Summary: `Identical to [filterByKey](#fn-filterByKey), except it returns only the first ` +
					`object in the resulting array instead of the whole array.`,
				Arguments: []funcArg{
					{
						Name:        `array`,
						Type:        `array`,
						Description: `The array of objects to filter.`,
					}, {
						Name:        `key`,
						Type:        `string`,
						Description: `The name of the key on each object in the given array to check the value of.`,
					}, {
						Name: `expression`,
						Type: `string`,
						Description: `The "{{ expression }}" to apply to the value at key from each object.  ` +
							`Uses the same expression rules as [filter](#fn-filter)`,
					},
				},
				Examples: []funcExample{
					{
						Code: `firstByKey [{"id": "a", "value": 1}, {"id": "b", "value": 1}, {"id": "c", "value": 2}] 1`,
						Return: map[string]interface{}{
							"id":    "a",
							"value": 1,
						},
					},
				},
				Function: func(input interface{}, key string, exprs ...interface{}) (interface{}, error) {
					if v, err := filterByKey(funcs, input, key, exprs...); err == nil {
						return sliceutil.First(v), nil
					} else {
						return nil, err
					}
				},
			}, {
				Name: `transformValues`,
				Summary: `Return all elements of the given array of objects with the value at a key transformed ` +
					`by the given expression.`,
				Arguments: []funcArg{
					{
						Name:        `array`,
						Type:        `array`,
						Description: `The array of objects to filter.`,
					}, {
						Name:        `key`,
						Type:        `string`,
						Description: `The name of the key on each object in the given array to modify.`,
					}, {
						Name: `expression`,
						Type: `string`,
						Description: `The "{{ expression }}" to apply to the value at key from each object.  ` +
							`Uses the same expression rules as [filter](#fn-filter)`,
					},
				},
				Examples: []funcExample{
					{
						Code: `transformValues [{"name": "alice"}, {"name": "mallory"}, {"name": "bob"}] "name" "{{ upper . }}"`,
						Return: []map[string]interface{}{
							{"name": `ALICE`},
							{"name": `MALLORY`},
							{"name": `BOB`},
						},
					},
				},
				Function: func(input interface{}, key string, expr string) ([]interface{}, error) {
					var out = make([]interface{}, 0)

					for i, obj := range sliceutil.Sliceify(input) {
						var tmpl = NewTemplate(`inline`, TextEngine)
						tmpl.Funcs(funcs)
						var m = maputil.M(obj)

						if !strings.HasPrefix(expr, `{{`) {
							expr = `{{` + expr
						}

						if !strings.HasSuffix(expr, `}}`) {
							expr = expr + `}}`
						}

						if err := tmpl.ParseString(expr); err == nil {
							var output = bytes.NewBuffer(nil)
							var value = m.Auto(key)

							if err := tmpl.Render(output, value, ``); err == nil {
								var evalValue = stringutil.Autotype(output.String())
								m.Set(key, evalValue)
								out = append(out, m.MapNative())
							} else {
								return nil, fmt.Errorf("item %d: %v", i, err)
							}
						} else {
							return nil, fmt.Errorf("failed to parse template: %v", err)
						}
					}

					return out, nil
				},
			}, {
				Name: `uniqByKey`,
				Summary: `Return an array of objects containing only unique entries in the given array of objects. ` +
					`Uniqueness is determined by comparing the values at the given key for each object.  The first ` +
					`time a value is encountered, that value's parent object is included in the output.  All ` +
					`subsequent objects with the same value at that key will be discarded.`,
				Arguments: []funcArg{
					{
						Name:        `array`,
						Type:        `array`,
						Description: `The array of objects to filter.`,
					}, {
						Name:        `key`,
						Type:        `string`,
						Description: `The name of the key on each object to consider for determining uniqueness.`,
					}, {
						Name:     `expression`,
						Type:     `string`,
						Optional: true,
						Description: `The "{{ expression }}" to apply to the value at key from each object before determining uniqueness.  ` +
							`Uses the same expression rules as [filter](#fn-filter)`,
					},
				},
				Examples: []funcExample{
					{
						Code: `uniqueByKey [{"id": "a", "value": 1}, {"id": "b", "value": 1}, {"id": "c", "value": 2}] "value"`,
						Return: []map[string]interface{}{
							{"id": "a", "value": 1},
							{"id": "c", "value": 2},
						},
					}, {
						Description: `Here we provide an expression that will normalize the value of the "name" field before performing the unique operation.`,
						Code:        `uniqueByKey [{"name": "bob", "i": 1}, {"name": "BOB", "i": 2}, {"name": "Bob", "i": 3}] "name" "{{ upper . }}"`,
						Return: []map[string]interface{}{
							{"name": "BOB", "i": 1},
						},
					},
				},
				Function: func(input interface{}, key string, exprs ...interface{}) ([]interface{}, error) {
					return uniqByKey(funcs, input, key, false, exprs...)
				},
			}, {
				Name: `uniqByKeyLast`,
				Summary: `Identical to [uniqByKey](#fn-uniqByKey), except the _last_ of a set of objects grouped ` +
					`by key is included in the output, not the first.`,
				Arguments: []funcArg{
					{
						Name:        `array`,
						Type:        `array`,
						Description: `The array of objects to filter.`,
					}, {
						Name:        `key`,
						Type:        `string`,
						Description: `The name of the key on each object to consider for determining uniqueness.`,
					}, {
						Name:     `expression`,
						Type:     `string`,
						Optional: true,
						Description: `The "{{ expression }}" to apply to the value at key from each object before determining uniqueness.  ` +
							`Uses the same expression rules as [filter](#fn-filter)`,
					},
				},
				Examples: []funcExample{
					{
						Code: `uniqByKeyLast [{"id": "a", "value": 1}, {"id": "b", "value": 1}, {"id": "c", "value": 2}] "value"`,
						Return: []map[string]interface{}{
							{"id": "b", "value": 1},
							{"id": "c", "value": 2},
						},
					}, {
						Description: `Here we provide an expression that will normalize the value of the "name" field before performing the unique operation.`,
						Code:        `uniqueByKey [{"name": "bob", "i": 1}, {"name": "BOB", "i": 2}, {"name": "Bob", "i": 3}] "name" "{{ upper . }}"`,
						Return: []map[string]interface{}{
							{"name": "BOB", "i": 3},
						},
					},
				},
				Function: func(input interface{}, key string, exprs ...interface{}) ([]interface{}, error) {
					return uniqByKey(funcs, input, key, true, exprs...)
				},
			}, {

				Name:    `sortByKey`,
				Summary: `Sort the given array of objects by comparing the values of the given key for all objects.`,
				Arguments: []funcArg{
					{
						Name:        `array`,
						Type:        `array`,
						Description: `The array of objects to sort.`,
					}, {
						Name:        `key`,
						Type:        `string`,
						Description: `The name of the key on each object whose values should determine the order of the output array.`,
					}, {
						Name:     `expression`,
						Type:     `string`,
						Optional: true,
						Description: `The "{{ expression }}" to apply to the value at key from each object before determining uniqueness.  ` +
							`Uses the same expression rules as [filter](#fn-filter)`,
					},
				},
				Examples: []funcExample{
					{
						Code: `sortByKey [{"name": "bob"}, {"name": "Mallory"}, {"name": "ALICE"}] "name"`,
						Return: []map[string]interface{}{
							{"name": "ALICE"},
							{"name": "bob"},
							{"name": "Mallory"},
						},
					},
				},
				Function: func(input interface{}, key string) ([]interface{}, error) {
					var out = sliceutil.Sliceify(input)
					sort.Slice(out, func(i int, j int) bool {
						var mI = maputil.M(out[i])
						var mJ = maputil.M(out[j])
						return mI.String(key) < mJ.String(key)
					})
					return out, nil
				},
			}, {

				Name:    `rSortByKey`,
				Summary: `Same as sortByKey, but reversed.`,
				Arguments: []funcArg{
					{
						Name:        `array`,
						Type:        `array`,
						Description: `The array of objects to sort.`,
					}, {
						Name:        `key`,
						Type:        `string`,
						Description: `The name of the key on each object whose values should determine the order of the output array.`,
					}, {
						Name:     `expression`,
						Type:     `string`,
						Optional: true,
						Description: `The "{{ expression }}" to apply to the value at key from each object before determining uniqueness.  ` +
							`Uses the same expression rules as [filter](#fn-filter)`,
					},
				},
				Examples: []funcExample{
					{
						Code: `rSortByKey [{"name": "bob"}, {"name": "Mallory"}, {"name": "ALICE"}] "name"`,
						Return: []map[string]interface{}{
							{"name": "Mallory"},
							{"name": "bob"},
							{"name": "ALICE"},
						},
					},
				},
				Function: func(input interface{}, key string) ([]interface{}, error) {
					var out = sliceutil.Sliceify(input)
					sort.Slice(out, func(i int, j int) bool {
						var mI = maputil.M(out[i])
						var mJ = maputil.M(out[j])
						return mI.String(key) > mJ.String(key)
					})
					return out, nil
				},
			}, {
				Name:    `isortByKey`,
				Summary: `Sort the given array of objects by comparing the values of the given key for all objects (case-insensitive compare).`,
				Arguments: []funcArg{
					{
						Name:        `array`,
						Type:        `array`,
						Description: `The array of objects to sort.`,
					}, {
						Name:        `key`,
						Type:        `string`,
						Description: `The name of the key on each object whose values should determine the order of the output array.`,
					}, {
						Name:     `expression`,
						Type:     `string`,
						Optional: true,
						Description: `The "{{ expression }}" to apply to the value at key from each object before determining uniqueness.  ` +
							`Uses the same expression rules as [filter](#fn-filter)`,
					},
				},
				Examples: []funcExample{
					{
						Code: `isortByKey [{"name": "Bob"}, {"name": "Mallory"}, {"name": "Alice"}] "name"`,
						Return: []map[string]interface{}{
							{"name": "Alice"},
							{"name": "Bob"},
							{"name": "Mallory"},
						},
					},
				},
				Function: func(input interface{}, key string) ([]interface{}, error) {
					var out = sliceutil.Sliceify(input)
					sort.Slice(out, func(i int, j int) bool {
						var mI = maputil.M(out[i])
						var mJ = maputil.M(out[j])

						return strings.ToLower(mI.String(key)) < strings.ToLower(mJ.String(key))
					})
					return out, nil
				},
			}, {

				Name:    `irSortByKey`,
				Summary: `Same as isortByKey, but reversed (case-insensitive compare).`,
				Arguments: []funcArg{
					{
						Name:        `array`,
						Type:        `array`,
						Description: `The array of objects to sort.`,
					}, {
						Name:        `key`,
						Type:        `string`,
						Description: `The name of the key on each object whose values should determine the order of the output array.`,
					}, {
						Name:     `expression`,
						Type:     `string`,
						Optional: true,
						Description: `The "{{ expression }}" to apply to the value at key from each object before determining uniqueness.  ` +
							`Uses the same expression rules as [filter](#fn-filter)`,
					},
				},
				Examples: []funcExample{
					{
						Code: `irSortByKey [{"name": "Bob"}, {"name": "Mallory"}, {"name": "Alice"}] "name"`,
						Return: []map[string]interface{}{
							{"name": "Mallory"},
							{"name": "Bob"},
							{"name": "Alice"},
						},
					},
				},
				Function: func(input interface{}, key string) ([]interface{}, error) {
					var out = sliceutil.Sliceify(input)
					sort.Slice(out, func(i int, j int) bool {
						var mI = maputil.M(out[i])
						var mJ = maputil.M(out[j])
						return strings.ToLower(mI.String(key)) > strings.ToLower(mJ.String(key))
					})
					return out, nil
				},
			}, {
				Name:    `pluck`,
				Summary: `Retrieve a value at the given key from each object in a given array of objects.`,
				Arguments: []funcArg{
					{
						Name:        `array`,
						Type:        `array`,
						Description: `The array of objects to retrieve values from.`,
					}, {
						Name:        `key`,
						Type:        `string`,
						Description: `The name of the key on each object whose values should returned.`,
					}, {
						Name:        `additional_keys`,
						Type:        `strings`,
						Optional:    true,
						Variadic:    true,
						Description: `If specified, the values of these additional keys will be appended (in order) to the output array.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `pluck [{"name": "Bob"}, {"name": "Mallory"}, {"name": "Alice"}] "name"`,
						Return: []string{`Bob`, `Mallory`, `Alice`},
					},
				},
				Function: func(input interface{}, key string, additionalKeys ...interface{}) []interface{} {
					var out = maputil.Pluck(input, strings.Split(key, `.`))

					for _, ak := range sliceutil.Stringify(additionalKeys) {
						out = append(out, maputil.Pluck(input, strings.Split(ak, `.`))...)
					}

					return out
				},
			}, {
				Name:    `keys`,
				Summary: `Return an array of key names specifying all the keys of the given object.`,
				Arguments: []funcArg{
					{
						Name:        `object`,
						Type:        `object`,
						Description: `The object to return the key names from.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `keys {"id": "a", "value": 1}`,
						Return: []string{`id`, `value`},
					},
				},
				Function: func(input interface{}) []interface{} {
					return maputil.Keys(input)
				},
			}, {
				Name:    `values`,
				Summary: `Return an array of values from the given object.`,
				Arguments: []funcArg{
					{
						Name:        `object`,
						Type:        `object`,
						Description: `The object to return values from.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `values {"id": "a", "value": 1}`,
						Return: []interface{}{`a`, 1},
					},
				},
				Function: func(input interface{}) []interface{} {
					return maputil.MapValues(input)
				},
			}, {
				Name: `get`,
				Summary: `Retrieve a value from a given object.  Key can be specified as a dot.separated.list string or ` +
					`array of keys that describes a path from the given object, through any intermediate nested objects, ` +
					`down to the object containing the desired value.`,
				Arguments: []funcArg{
					{
						Name:        `object`,
						Type:        `object`,
						Description: `The object to retrieve the value from`,
					}, {
						Name:        `key`,
						Type:        `string, array`,
						Description: `The key name, path, or array of values representing path segments pointing to the value to retrieve.`,
					}, {
						Name:     `fallback`,
						Type:     `any`,
						Optional: true,
						Description: `If the value at the given key does not exist, this value will be returned instead.  ` +
							`If not specified, the default return value is null.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `get {"name: "Bob"} "name"`,
						Return: `Bob`,
					},
					{
						Code:   `get {"properties": {"info": {"name: "Bob"}}} "properties.info.name"`,
						Return: `Bob`,
					},
					{
						Code:   `get {"properties": {"info": {"name: "Bob"}}} "properties.info.age"`,
						Return: nil,
					}, {
						Code:   `get {"properties": {"info": {"name: "Bob"}}} "properties.info.age" 42`,
						Return: 42,
					},
					{
						Code:   `get {"properties": {"info.name": "Bob"}} ["properties", "info.name"]`,
						Return: `Bob`,
					},
				},
				Function: func(input interface{}, key interface{}, fallback ...interface{}) interface{} {
					var fb interface{}

					if len(fallback) > 0 {
						fb = fallback[0]
					}

					var split []string

					if typeutil.IsArray(key) {
						split = sliceutil.Stringify(key)
					} else {
						split = strings.Split(typeutil.String(key), `.`)
					}

					return maputil.DeepGet(input, split, fb)
				},
			}, {
				Name: `set`,
				Summary: `Set a key on a given object to a value. Key can be specified as a dot.separated.list string or ` +
					`array of keys that describes a path in the given object, through any intermediate nested objects, ` +
					`down to the object where the given value will go.`,
				Arguments: []funcArg{
					{
						Name:        `object`,
						Type:        `object`,
						Description: `The object to retrieve the value from`,
					}, {
						Name:        `key`,
						Type:        `string, array`,
						Description: `The key name, path, or array of values representing path segments pointing to the value to create or modify.`,
					}, {
						Name:        `value`,
						Type:        `any`,
						Description: `The value to set.`,
					},
				},
				Function: func(input interface{}, key interface{}, value interface{}) error {
					var split []string

					if typeutil.IsArray(key) {
						split = sliceutil.Stringify(key)
					} else {
						split = strings.Split(typeutil.String(key), `.`)
					}

					maputil.DeepSet(input, split, value)
					return nil
				},
			}, {
				Name: `findKey`,
				Aliases: []string{
					`findkey`,
				},
				Summary: `Recursively scans the given array or object and returns all values of the given key.`,
				Arguments: []funcArg{
					{
						Name:        `input`,
						Type:        `array, object`,
						Description: `The object or array of object to retrieve values from.`,
					}, {
						Name:        `key`,
						Type:        `string`,
						Description: `The name of the key in any objects encountered whose value should be included in the output.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `findKey [{"id": 1, "children": [{"id": 3}, {"id": 5}, {"id": 8}]} "id"`,
						Return: []int{1, 3, 5, 8},
					},
				},
				Function: func(input interface{}, key string) ([]interface{}, error) {
					var values = make([]interface{}, 0)

					if err := maputil.Walk(input, func(value interface{}, path []string, isLeaf bool) error {
						if isLeaf && path[len(path)-1] == key {
							values = append(values, value)
						}

						return nil
					}); err != nil {
						return nil, err
					}

					return values, nil
				},
			}, {
				Name:    `has`,
				Summary: `Return whether a specific element is in an array.`,
				Arguments: []funcArg{
					{
						Name:        `wanted`,
						Type:        `any`,
						Description: `The value being sought out.`,
					}, {
						Name:        `input`,
						Type:        `array`,
						Description: `The array to search within.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `has "e" ["a", "e", "i", "o", "u"]`,
						Return: true,
					}, {
						Code:   `has "y" ["a", "e", "i", "o", "u"]`,
						Return: false,
					}, {
						Code:   `has "13" ["3", "5", "8", "13"]`,
						Return: true,
					}, {
						Code:   `has 13 ["3", "5", "8", "13"]`,
						Return: true,
					}, {
						Code:   `has 14 ["3", "5", "8", "13"]`,
						Return: false,
					},
				},
				Function: func(want interface{}, input interface{}) bool {
					for _, have := range sliceutil.Sliceify(input) {
						if eq, err := stringutil.RelaxedEqual(have, want); err == nil && eq == true {
							return true
						}
					}

					return false
				},
			}, {
				Name:    `any`,
				Summary: `Return whether an array contains any of a set of desired items.`,
				Arguments: []funcArg{
					{
						Name:        `input`,
						Type:        `array`,
						Description: `The array to search within.`,
					}, {
						Name:        `wanted`,
						Type:        `any`,
						Variadic:    true,
						Description: `A list of values, any of which being present in the given array will return true.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `any ["a", "e", "i", "o", "u"] "e" "y" "x"`,
						Return: true,
					},
					{
						Code:   `any ["r", "s", "t", "l", "n", "e"] "f" "m" "w" "o"`,
						Return: false,
					},
				},
				Function: func(input interface{}, wants ...interface{}) bool {
					for _, have := range sliceutil.Sliceify(input) {
						for _, want := range wants {
							if eq, err := stringutil.RelaxedEqual(have, want); err == nil && eq == true {
								return true
							}
						}
					}

					return false
				},
			}, {
				Name:    `indexOf`,
				Summary: `Iterate through an array and return the index of a given value, or -1 if not present.`,
				Arguments: []funcArg{
					{
						Name:        `input`,
						Type:        `array`,
						Description: `The array to search within.`,
					}, {
						Name:        `wanted`,
						Type:        `any`,
						Description: `The value being sought out.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `indexOf ["a", "e", "i", "o", "u"] "e"`,
						Return: 1,
					},
					{
						Code:   `indexOf ["a", "e", "i", "o", "u"] "y"`,
						Return: -1,
					},
				},
				Function: func(slice interface{}, value interface{}) (index int) {
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
				},
			}, {
				Name:    `slice`,
				Summary: `Return a subset of the given array.  Items in an array are counted starting from zero.`,
				Arguments: []funcArg{
					{
						Name:        `input`,
						Type:        `array`,
						Description: `The array to slice up.`,
					}, {
						Name: `from`,
						Type: `integer`,
						Description: `The starting index within the given array to start returning items from.  ` +
							`Can be negative, indicating the nth element from the end of the array (e.g: -1 means ` +
							`"last element", -2 is "second from last", and so on.).`,
					}, {
						Name: `to`,
						Type: `integer`,
						Description: `The end index within the given array to stop returning items from.  ` +
							`Can be negative, indicating the nth element from the end of the array (e.g: -1 means ` +
							`"last element", -2 is "second from last", and so on.).`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `slice ["a", "e", "i", "o", "u"] 0 -1`,
						Return: []string{`a`, `e`, `i`, `o`, `u`},
					}, {
						Code:   `slice ["a", "e", "i", "o", "u"] 2 -1`,
						Return: []string{`i`, `o`, `u`},
					}, {
						Code:   `slice ["a", "e", "i", "o", "u"] -3 -1`,
						Return: []string{`i`, `o`, `u`},
					}, {
						Code:   `slice ["a", "e", "i", "o", "u"] 1 1`,
						Return: []string{`e`},
					},
				},
				Function: func(slice interface{}, from interface{}, to interface{}) []interface{} {
					return sliceutil.Slice(slice, int(typeutil.Int(from)), int(typeutil.Int(to)))
				},
			}, {
				Name:    `sslice`,
				Summary: `Identical to [slice](#fn-slice), but returns an array of strings.`,
				Function: func(slice interface{}, from interface{}, to interface{}) []string {
					return sliceutil.StringSlice(slice, int(typeutil.Int(from)), int(typeutil.Int(to)))
				},
			}, {
				Name:    `uniq`,
				Summary: `Return an array containing only unique values from the given array.`,
				Arguments: []funcArg{
					{
						Name:        `input`,
						Type:        `array`,
						Description: `The array to unique.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `uniq ["a", "a", "b", "b", "b", "c"]`,
						Return: []string{`a`, `b`, `c`},
					},
				},
				Function: func(slice interface{}) []interface{} {
					return sliceutil.Unique(slice)
				},
			}, {
				Name:    `flatten`,
				Summary: `Return an array of values with all nested arrays collapsed down to a single, flat array.`,
				Arguments: []funcArg{
					{
						Name:        `input`,
						Type:        `array`,
						Description: `The array to flatten.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `flatten ["a", ["a", "b"], ["b", "b", ["c"]]]`,
						Return: []string{`a`, `a`, `b`, `b`, `b`, `c`},
					},
				},
				Function: func(slice interface{}) []interface{} {
					return sliceutil.Flatten(slice)
				},
			}, {
				Name:    `compact`,
				Summary: `Return an copy of given array with all empty, null, and zero elements removed.`,
				Arguments: []funcArg{
					{
						Name:        `input`,
						Type:        `array`,
						Description: `The array to compact.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `uniq ["a", null, "b", 0, false, "c"]`,
						Return: []string{`a`, `b`, `c`},
					},
				},
				Function: func(slice interface{}) []interface{} {
					return sliceutil.Compact(slice)
				},
			}, {
				Name:    `first`,
				Summary: `Return the first value from the given array, or null if the array is empty.`,
				Arguments: []funcArg{
					{
						Name:        `input`,
						Type:        `array`,
						Description: `The array to read from.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `first ["a", "b", "c", "d"]`,
						Return: `a`,
					},
				},
				Function: func(slice interface{}) (out interface{}, err error) {
					err = sliceutil.Each(slice, func(i int, value interface{}) error {
						out = value
						return sliceutil.Stop
					})

					return
				},
			}, {
				Name: `rest`,
				Summary: `Return all but the first value from the given array, or an empty array of the given ` +
					`array's length is <= 1.`,
				Arguments: []funcArg{
					{
						Name:        `input`,
						Type:        `array`,
						Description: `The array to read from.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `rest ["a", "b", "c", "d"]`,
						Return: []string{`b`, `c`, `d`},
					}, {
						Code:   `rest ["a"]`,
						Return: []string{},
					},
				},
				Function: func(slice interface{}) ([]interface{}, error) {
					return sliceutil.Rest(slice), nil
				},
			}, {
				Name:    `last`,
				Summary: `Return the last value from the given array, or null if the array is empty.`,
				Arguments: []funcArg{
					{
						Name:        `input`,
						Type:        `array`,
						Description: `The array to read from.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `last ["a", "b", "c", "d"]`,
						Return: `d`,
					},
				},
				Function: func(slice interface{}) (out interface{}, err error) {
					err = sliceutil.Each(slice, func(i int, value interface{}) error {
						out = value
						return nil
					})

					return
				},
			}, {
				Name:    `count`,
				Summary: `Identical to the built-in "len" function, but is less picky about types.`,
				Arguments: []funcArg{
					{
						Name:        `input`,
						Type:        `array`,
						Description: `The array to read from.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `count ["a", "b", "c", "d"]`,
						Return: 4,
					},
				},
				Function: func(in interface{}) int {
					return sliceutil.Len(in)
				},
			}, {
				Name:    `sort`,
				Summary: `Return an array sorted in lexical ascending order.`,
				Arguments: []funcArg{
					{
						Name:        `input`,
						Type:        `array`,
						Description: `The array to sort.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `sort ["d", "a", "c", "b"]`,
						Return: []string{`a`, `b`, `c`, `d`},
					},
				},
				Function: func(input interface{}) []interface{} {
					var out = sliceutil.Sliceify(input)

					sort.Slice(out, func(i, j int) bool {
						var iv = typeutil.String(out[i])
						var jv = typeutil.String(out[j])

						return iv < jv
					})

					return out
				},
			}, {
				Name:    `rsort`,
				Summary: `Return the array sorted in lexical descending order.`,
				Arguments: []funcArg{
					{
						Name:        `input`,
						Type:        `array`,
						Description: `The array to sort.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `rsort ["d", "a", "c", "b"]`,
						Return: []string{`d`, `c`, `b`, `a`},
					},
				},
				Function: func(input interface{}) []interface{} {
					var out = sliceutil.Sliceify(input)

					sort.Slice(out, func(i, j int) bool {
						var iv = typeutil.String(out[i])
						var jv = typeutil.String(out[j])

						return iv > jv
					})

					return out
				},
			}, {
				Name:    `isort`,
				Summary: `Return an array sorted in lexical ascending order (case-insensitive.)`,
				Arguments: []funcArg{
					{
						Name:        `input`,
						Type:        `array`,
						Description: `The array to sort.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `isort ["bob", "ALICE", "Mallory"]`,
						Return: []string{`ALICE`, `bob`, `Mallory`},
					},
				},
				Function: func(input interface{}) []interface{} {
					var out = sliceutil.Sliceify(input)

					sort.Slice(out, func(i, j int) bool {
						var iv = strings.ToLower(typeutil.String(out[i]))
						var jv = strings.ToLower(typeutil.String(out[j]))

						return iv < jv
					})

					return out
				},
			}, {
				Name:    `irsort`,
				Summary: `Return the array sorted in lexical descending order (case-insensitive.)`,
				Arguments: []funcArg{
					{
						Name:        `input`,
						Type:        `array`,
						Description: `The array to sort.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `irsort ["bob", "ALICE", "Mallory"]`,
						Return: []string{`Mallory`, `bob`, `ALICE`},
					},
				},
				Function: func(input interface{}, keys ...string) []interface{} {
					var out = sliceutil.Sliceify(input)

					sort.Slice(out, func(i, j int) bool {
						var iv = strings.ToLower(typeutil.String(out[i]))
						var jv = strings.ToLower(typeutil.String(out[j]))

						return iv > jv
					})

					return out
				},
			}, {
				Name:    `mostcommon`,
				Summary: `Return the element in a given array that appears the most frequently.`,
				Arguments: []funcArg{
					{
						Name:        `input`,
						Type:        `array`,
						Description: `The array to read from.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `mostcommon ["a", "a", "b", "b", "b", "c"]`,
						Return: `b`,
					},
				},
				Function: func(slice interface{}) (interface{}, error) {
					return commonses(slice, `most`)
				},
			}, {
				Name:    `leastcommon`,
				Summary: `Return the element in a given array that appears the least frequently.`,
				Arguments: []funcArg{
					{
						Name:        `input`,
						Type:        `array`,
						Description: `The array to read from.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `leastcommon ["a", "a", "b", "b", "b", "c"]`,
						Return: `c`,
					},
				},
				Function: func(slice interface{}) (interface{}, error) {
					return commonses(slice, `least`)
				},
			}, {
				Name: `sliceify`,
				Summary: `Convert the given input into an array.  If the value is already an array, ` +
					`this just returns that array.  Otherwise, it returns an array containing the ` +
					`given value as its only element.`,
				Arguments: []funcArg{
					{
						Name:        `input`,
						Type:        `any`,
						Description: `The value to make into an array.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `sliceify ["a", "b", "c"]`,
						Return: []string{`a`, `b`, `c`},
					},
					{
						Code:   `sliceify 4`,
						Return: []int{4},
					},
				},
				Function: func(slice interface{}) []interface{} {
					return sliceutil.Sliceify(slice)
				},
			}, {
				Name: `stringify`,
				Summary: `Identical to [sliceify](#fn-sliceify), but converts all values to ` +
					`strings and returns an array of strings.`,
				Function: func(slice interface{}) []string {
					return sliceutil.Stringify(slice)
				},
			}, {
				Name:    `intersect`,
				Summary: `Return the intersection of two arrays.`,
				Arguments: []funcArg{
					{
						Name:        `first`,
						Type:        `array`,
						Description: `The first array.`,
					}, {
						Name:        `second`,
						Type:        `array`,
						Description: `The second array.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `intersect ["b", "a", "c"] ["c", "b", "d"]`,
						Return: []string{`b`, `c`},
					},
					{
						Code:   `intersect ["a", "b", "c"] ["x", "y", "z"]`,
						Return: []string{},
					},
				},
				Function: func(first interface{}, second interface{}) []interface{} {
					return sliceutil.Intersect(first, second)
				},
			}, {
				Name:    `difference`,
				Summary: `Return the first array with common elements from the second removed.`,
				Arguments: []funcArg{
					{
						Name:        `first`,
						Type:        `array`,
						Description: `The first array.`,
					}, {
						Name:        `second`,
						Type:        `array`,
						Description: `The second array.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `difference ["b", "a", "c"] ["c", "b", "d"]`,
						Return: []string{`a`},
					},
					{
						Code:   `difference ["a", "b", "c"] ["x", "y", "z"]`,
						Return: []string{`a`, `b`, `c`},
					},
				},
				Function: func(first interface{}, second interface{}) []interface{} {
					return sliceutil.Difference(first, second)
				},
			}, {
				Name:    `mapify`,
				Summary: `Return the given value returned as a rangeable object.`,
				Function: func(input interface{}) map[string]interface{} {
					return maputil.DeepCopy(input)
				},
			}, {
				Name:    `onlyKeys`,
				Summary: `Return the given object with only the specified keys included.`,
				Arguments: []funcArg{
					{
						Name:        `in`,
						Type:        `object`,
						Description: `The object to filter.`,
					}, {
						Name:        `keys`,
						Type:        `string`,
						Description: `Zero or more keys to include in the output.`,
						Optional:    true,
						Variadic:    true,
					},
				},
				Examples: []funcExample{
					{
						Code: `onlyKeys {"a": 1, "b": 2, "c": 3} "a" "c"`,
						Return: map[string]interface{}{
							`a`: 1,
							`c`: 3,
						},
					},
				},
				Function: func(input interface{}, keys ...string) map[string]interface{} {
					var out = maputil.DeepCopy(input)

					for k, _ := range out {
						if !sliceutil.ContainsString(keys, k) {
							delete(out, k)
						}
					}

					return out
				},
			}, {
				Name:    `exceptKeys`,
				Summary: `Return the given object with the specified keys removed.`,
				Arguments: []funcArg{
					{
						Name:        `in`,
						Type:        `object`,
						Description: `The object to filter.`,
					}, {
						Name:        `keys`,
						Type:        `string`,
						Description: `Zero or more keys to exclude from the output.`,
						Optional:    true,
						Variadic:    true,
					},
				},
				Examples: []funcExample{
					{
						Code: `exceptKeys {"a": 1, "b": 2, "c": 3} "a" "c"`,
						Return: map[string]interface{}{
							`b`: 2,
						},
					},
				},
				Function: func(input interface{}, keys ...interface{}) map[string]interface{} {
					var out = maputil.DeepCopy(input)
					keys = sliceutil.Flatten(keys)

					for _, key := range sliceutil.Stringify(keys) {
						delete(out, key)
					}

					return out
				},
			}, {
				Name: `groupBy`,
				Summary: `Return the given array of objects as a grouped object, keyed on the ` +
					`value of the specified group field. The field argument can be an ` +
					`expression that receives the value and returns a transformed version of it.`,
				Arguments: []funcArg{
					{
						Name:        `array`,
						Type:        `array`,
						Description: `An array of objects to group.`,
					}, {
						Name:        `key`,
						Type:        `string`,
						Description: `The key to retreive from each object, the value of which will determine the group names.`,
					},
				},
				Examples: []funcExample{
					{
						Code: `groupBy [{"name": "Bob", "title": "Friend"}, {"name": "Mallory", "title": "Foe"}, {"name": "Alice", "title": "Friend"}] "title"`,
						Return: map[string][]interface{}{
							`Friend`: []interface{}{
								map[string]interface{}{
									`name`:  `Bob`,
									`title`: `Friend`,
								},
								map[string]interface{}{
									`name`:  `Alice`,
									`title`: `Friend`,
								},
							},
							`Foe`: []interface{}{
								map[string]interface{}{
									`name`:  `Mallory`,
									`title`: `Foe`,
								},
							},
						},
					},
				},
				Function: func(sliceOfMaps interface{}, key string, tpls ...interface{}) (map[string][]interface{}, error) {
					if !typeutil.IsArray(sliceOfMaps) {
						return nil, fmt.Errorf("groupBy only works on arrays of objects, got %T", sliceOfMaps)
					}

					var output = make(map[string][]interface{})
					var valueTpls = sliceutil.Stringify(tpls)

					if items := sliceutil.Sliceify(sliceOfMaps); len(items) > 0 {
						if !typeutil.IsMap(items[0]) {
							return nil, fmt.Errorf("groupBy only works on arrays of objects, got %T", items[0])
						}

						for _, item := range items {
							var value = maputil.DeepGet(item, strings.Split(key, `.`))

							if len(valueTpls) > 0 && valueTpls[0] != `` {
								if stringutil.IsSurroundedBy(valueTpls[0], `{{`, `}}`) {
									var tmpl = NewTemplate(`inline`, TextEngine)
									tmpl.Funcs(funcs)

									if err := tmpl.ParseString(valueTpls[0]); err == nil {
										var output = bytes.NewBuffer(nil)

										if err := tmpl.Render(output, value, ``); err == nil {
											value = stringutil.Autotype(output.String())
										} else {
											return nil, fmt.Errorf("Key Template failed: %v", err)
										}
									} else {
										return nil, fmt.Errorf("Failed to parse Key template: %v", err)
									}
								}
							}

							var valueS = fmt.Sprintf("%v", value)

							if v, ok := output[valueS]; ok {
								output[valueS] = append(v, item)
							} else {
								output[valueS] = []interface{}{item}
							}
						}
					}

					return output, nil
				},
			}, {
				Name:    `head`,
				Summary: `Return the first _n_ items from an array.`,
				Arguments: []funcArg{
					{
						Name:        `input`,
						Type:        `array`,
						Description: `The array to read from.`,
					}, {
						Name:        `count`,
						Type:        `integer`,
						Description: `The number of items to retrieve from the beginning of the array.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `head ["a", "b", "c", "d"] 2`,
						Return: []string{`a`, `b`},
					},
				},
				Function: func(input interface{}, n int) []interface{} {
					if typeutil.IsZero(input) {
						return make([]interface{}, 0)
					}

					var items = sliceutil.Sliceify(input)

					if len(items) < n {
						return items
					} else {
						return items[0:n]
					}
				},
			}, {
				Name:    `tail`,
				Summary: `Return the last _n_ items from an array.`,
				Arguments: []funcArg{
					{
						Name:        `input`,
						Type:        `array`,
						Description: `The array to read from.`,
					}, {
						Name:        `count`,
						Type:        `integer`,
						Description: `The number of items to retrieve from the end of the array.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `tail ["a", "b", "c", "d"] 2`,
						Return: []string{`c`, `d`},
					},
				},
				Function: func(input interface{}, n int) []interface{} {
					if typeutil.IsZero(input) {
						return make([]interface{}, 0)
					}

					var items = sliceutil.Sliceify(input)

					if len(items) < n {
						return items
					} else {
						return items[len(items)-n:]
					}
				},
			}, {
				Name:    `shuffle`,
				Summary: `Return the array with the elements rearranged in random order.`,
				Arguments: []funcArg{
					{
						Name:        `input`,
						Type:        `array`,
						Description: `The array to shuffle.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `shuffle ["a", "b", "c", "d"]`,
						Return: []string{`d`, `c`, `b`, `a`},
					},
				},
				Function: func(input ...interface{}) []interface{} {
					if typeutil.IsZero(input) {
						return make([]interface{}, 0)
					}

					var inputS = sliceutil.Sliceify(input)

					inputS = sliceutil.Flatten(inputS)

					for i := range inputS {
						var j = rand.Intn(i + 1)
						inputS[i], inputS[j] = inputS[j], inputS[i]
					}

					return inputS
				},
			}, {
				Name:    `shuffleInPlace`,
				Summary: `Shuffle the input array in place, returning the seed value used to shuffle the input.`,
				Arguments: []funcArg{
					{
						Name:        `input`,
						Type:        `array`,
						Description: `The array to shuffle.`,
					}, {
						Name:        `seed`,
						Type:        `int64`,
						Description: `An optional seed value to generate reproducible randomization.`,
						Optional:    true,
					},
				},
				Function: func(input interface{}, seeds ...int64) (int64, error) {
					var inlen = sliceutil.Len(input)
					var seed int64 = rand.Int63()

					if len(seeds) > 0 && seeds[0] != 0 {
						seed = seeds[0]
					}

					if typeutil.IsZero(input) {
						return seed, nil
					} else if inlen == 0 {
						return seed, nil
					} else if !typeutil.IsArray(input) {
						return seed, fmt.Errorf("input must be an array or slice")
					}

					var swap = reflect.Swapper(input)

					rand.New(rand.NewSource(seed)).Shuffle(
						inlen,
						swap,
					)

					return seed, nil
				},
			}, {
				Name: `apply`,
				Summary: `Apply a function to each of the elements in the given array. Note ` +
					`that functions must be unary (accept one argument of type _any_).`,
				Arguments: []funcArg{
					{
						Name:        `input`,
						Type:        `array`,
						Description: `The array to modify.`,
					}, {
						Name:        `functions`,
						Type:        `strings`,
						Variadic:    true,
						Description: `One or more functions to pass each element to. Only supports functions that accept a zero or one arguments.`,
					},
				},
				Examples: []funcExample{
					{
						Code:   `apply ["a", "B", "C", "d"] "upper"`,
						Return: []string{`A`, `B`, `C`, `D`},
					},
					{
						Code:   `apply ["a", "B", "C", "d"] "upper" "lower"`,
						Return: []string{`a`, `b`, `c`, `d`},
					},
				},
				Function: func(input interface{}, fns ...string) ([]interface{}, error) {
					var out = make([]interface{}, 0)

					if err := sliceutil.Each(input, func(i int, value interface{}) error {
						for _, fnName := range fns {
							switch fnName {
							case `apply`:
								return fmt.Errorf("nested %q is unsupported", "apply")
							}

							if fn, ok := funcs[fnName]; ok {
								if fnV := reflect.ValueOf(fn); fnV.Kind() == reflect.Func {
									var returns []reflect.Value

									switch nin := fnV.Type().NumIn(); nin {
									case 0:
										returns = fnV.Call([]reflect.Value{})
									case 1:
										returns = fnV.Call([]reflect.Value{
											reflect.ValueOf(value),
										})
									default:
										return fmt.Errorf("expected 0- or 1-argument function, %q takes %d arguments", fnName, nin)
									}

									switch len(returns) {
									case 2:
										// two-return functions must have a signature of (<something>, error)
										if lastT := returns[1].Type(); lastT.Implements(errorInterface) {
											value = returns[0].Interface()

											if v2 := returns[1].Interface(); v2 != nil {
												return fmt.Errorf("failed on %q: %v", fnName, v2.(error))
											}
										} else {
											return fmt.Errorf("last return value must be an error, got %v", lastT)
										}

									case 1:
										if lastT := returns[0].Type(); lastT.Implements(errorInterface) {
											if v1 := returns[0].Interface(); v1 != nil {
												return fmt.Errorf("failed on %q: %v", fnName, v1.(error))
											}
										} else {
											value = returns[0].Interface()
										}
									}
								} else {
									return fmt.Errorf("invalid function %q", fnName)
								}
							} else {
								return fmt.Errorf("unrecognized function %q", fnName)
							}
						}

						out = append(out, value)

						return nil
					}); err == nil {
						return out, nil
					} else {
						return nil, err
					}
				},
			}, {
				Name: `diffuse`,
				Summary: `Convert an array of objects or object representing a single-level ` +
					`hierarchy of items and expand it into a deeply-nested object.`,
				Arguments: []funcArg{
					{
						Name:        `input`,
						Type:        `array/object`,
						Description: `The array or object to expand into a deeply-nested object.`,
					}, {
						Name:        `joiner`,
						Type:        `string`,
						Variadic:    true,
						Description: `The string used in object keys that separates levels of the hierarchy.`,
					},
				},
				Examples: []funcExample{
					{
						Code: `diffuse {"properties/enabled": true, "properties/label": "items", "name": "Items", "properties/tasks/0": "do things", "properties/tasks/1": "do stuff"} "/"`,
						Return: map[string]interface{}{
							`name`: `Items`,
							`properties`: map[string]interface{}{
								`enabled`: true,
								`label`:   `items`,
								`tasks`: []interface{}{
									`do things`,
									`do stuff`,
								},
							},
						},
					},
				},
				Function: func(input interface{}, joiner string) (map[string]interface{}, error) {
					if in, err := prepCoalesceDiffuseInput(input); err == nil {
						return maputil.DiffuseMap(in, joiner)
					} else {
						return nil, fmt.Errorf("Can only diffuse arrays and objects, got %T", input)
					}
				},
			}, {
				Name: `coalesce`,
				Summary: `Convert an array of objects or object representing a deeply-nested ` +
					`hierarchy of items and collapse it into a flat (not nested) object.`,
				Arguments: []funcArg{
					{
						Name:        `input`,
						Type:        `array/object`,
						Description: `The array or object to expand into a deeply-nested object.`,
					}, {
						Name:        `joiner`,
						Type:        `string`,
						Variadic:    true,
						Description: `The string used in object keys that separates levels of the hierarchy.`,
					},
				},
				Function: func(input interface{}, joiner string) (map[string]interface{}, error) {
					if in, err := prepCoalesceDiffuseInput(input); err == nil {
						return maputil.CoalesceMap(in, joiner)
					} else {
						return nil, fmt.Errorf("Can only coalesce arrays and objects, got %T", input)
					}
				},
			}, {
				Name:    `isLastElement`,
				Summary: `Returns whether the given index in the given array is the last element in that array.`,
				Arguments: []funcArg{
					{
						Name:        `index`,
						Type:        `integer`,
						Description: `The current index of the item in the collection.`,
					}, {
						Name:        `array`,
						Type:        `array`,
						Description: `The array being checked`,
					},
				},
				Function: func(index interface{}, array interface{}) bool {
					var i = typeutil.Int(index)
					var arr = sliceutil.Sliceify(array)

					return (int(i) == (len(arr) - 1))
				},
			},
		},
	}

	group.Functions = append(group.Functions, []funcDef{
		{
			Name:     `findkey`,
			Alias:    `findKey`,
			Function: group.fn(`findKey`),
			Hidden:   true,
		},
	}...)

	return group
}

func prepCoalesceDiffuseInput(input interface{}) (map[string]interface{}, error) {
	var in = make(map[string]interface{})

	if typeutil.IsArray(input) {
		for i, v := range sliceutil.Stringify(input) {
			in[typeutil.String(i)] = v
		}
	} else if typeutil.IsMap(input) {
		in = typeutil.MapNative(input)
	} else {
		return nil, fmt.Errorf("Can only diffuse arrays and objects, got %T", input)
	}

	return in, nil
}
