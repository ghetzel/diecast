package internal

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

func loadStandardFunctionsCollections(funcs FuncMap, server ServerProxy) FuncGroup {
	var group = FuncGroup{
		Name: `Arrays and Objects`,
		Description: `For converting, modifying, and filtering arrays, objects, and arrays of ` +
			`objects. These functions are especially useful when working with data returned from Bindings.`,
		Functions: []FuncDef{
			{
				Name: `append`,
				Summary: `Append one or more values to the given array.  If the array given is not in fact an array, ` +
					`it will be converted into one, with the exception of null values, which will create an empty array.`,
				Arguments: []FuncArg{
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
				Examples: []FuncExample{
					{
						Input:  []string{"a", "b"},
						Code:   `append $.input "c" "d"`,
						Return: []string{`a`, `b`, `c`, `d`},
					},
				},
				Function: func(array any, values ...any) ([]any, error) {
					var out = make([]any, 0)

					if array != nil {
						out = append(out, sliceutil.Sliceify(array)...)
					}

					out = append(out, values...)

					return out, nil
				},
			}, {
				Name: `page`,
				Summary: `Returns an integer representing an offset used for accessing paginated values when ` +
					`given a page number and number of results per page.`,
				Arguments: []FuncArg{
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
				Examples: []FuncExample{
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
				Function: func(pagenum any, perpage any) int {
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
				Arguments: []FuncArg{
					{
						Name:        `array`,
						Type:        `array`,
						Description: `The array to reverse.`,
					},
				},
				Examples: []FuncExample{
					{
						Input:  []int{1, 2, 3},
						Code:   `reverse $.input`,
						Return: []int{3, 2, 1},
					},
				},
				Function: func(input any) []any {
					var array = sliceutil.Sliceify(input)
					var output = make([]any, len(array))

					for i := 0; i < len(array); i++ {
						output[len(array)-1-i] = array[i]
					}

					return output
				},
			}, {
				Name:    `filter`,
				Summary: `Return the given array with only elements where expression evaluates to a truthy value.`,
				Arguments: []FuncArg{
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
				Examples: []FuncExample{
					{
						Input:  []int{1, 2, 3, 4, 5},
						Code:   `filter $.input "{{ isOdd . }}"`,
						Return: []int{1, 3, 5},
					}, {
						Input: []map[string]any{
							{
								`active`: true,
								`a`:      1,
							},
							{
								`b`: 2,
							},
							{
								`active`: true,
								`c`:      3,
							},
						},
						Code: `filter $.input "{{ .active }}"`,
						Return: []map[string]any{
							{"active": true, "a": 1},
							{"active": true, "c": 3},
						},
					},
				},
				Function: func(input any, expr string) ([]any, error) {
					var out = make([]any, 0)

					for i, value := range sliceutil.Sliceify(input) {
						var tmpl = NewTemplateWithFuncs(`inline`, TextEngine, funcs)

						if !strings.HasPrefix(expr, `{{`) {
							expr = `{{ ` + expr
						}

						if !strings.HasSuffix(expr, `}}`) {
							expr = expr + ` }}`
						}

						if err := tmpl.ParseString(expr); err == nil {
							var output = bytes.NewBuffer(nil)

							if err := tmpl.Render(output, value, `inline`); err == nil {
								var outstr = output.String()

								switch outstr {
								case `<no value>`:
									outstr = ``
								}

								var evalValue = stringutil.Autotype(outstr)

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
				Arguments: []FuncArg{
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
				Examples: []FuncExample{
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
				Function: func(in any, expr string, negate ...bool) ([]string, error) {
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
				Name: `transformValues`,
				Summary: `Return all elements of the given array of objects with the value at a key transformed ` +
					`by the given expression.`,
				Arguments: []FuncArg{
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
				Examples: []FuncExample{
					{
						Input: []map[string]any{
							{
								"name": "alice",
							},
							{
								"name": "mallory",
							},
							{
								"name": "bob",
							},
						},
						Code: `transformValues $.input "name" "{{ upper . }}"`,
						Return: []map[string]any{
							{"name": `ALICE`},
							{"name": `MALLORY`},
							{"name": `BOB`},
						},
					},
				},
				Function: func(input any, key string, expr string) ([]any, error) {
					var out = make([]any, 0)

					for i, obj := range sliceutil.Sliceify(input) {
						var tmpl = NewTemplateWithFuncs(`inline`, TextEngine, funcs)
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

							if err := tmpl.Render(output, value, `inline`); err == nil {
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
				Arguments: []FuncArg{
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
				Examples: []FuncExample{
					{
						Input: []map[string]any{
							{
								"id":    "a",
								"value": 1,
							},
							{
								"id":    "b",
								"value": 1,
							},
							{
								"id":    "c",
								"value": 2,
							},
						},
						Code: `uniqByKey $.input "value"`,
						Return: []map[string]any{
							{"id": "a", "value": 1},
							{"id": "c", "value": 2},
						},
					}, {
						Description: `Here we provide an expression that will normalize the value of the "name" field before performing the unique operation.`,
						Input: []map[string]any{
							{
								"name": "bob",
								"i":    1,
							},
							{
								"name": "BOB",
								"i":    2,
							},
							{
								"name": "Bob",
								"i":    3,
							},
						},
						Code: `uniqByKey $.input "name" "{{ upper . }}"`,
						Return: []map[string]any{
							{"name": "bob", "i": 1},
						},
					},
				},
				Function: func(input any, key string, exprs ...any) ([]any, error) {
					return uniqByKey(funcs, input, key, false, exprs...)
				},
			}, {
				Name: `uniqByKeyLast`,
				Summary: `Identical to [uniqByKey](#fn-uniqByKey), except the _last_ of a set of objects grouped ` +
					`by key is included in the output, not the first.`,
				Arguments: []FuncArg{
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
				Examples: []FuncExample{
					{
						Input: []map[string]any{
							{
								"id":    "a",
								"value": 1,
							},
							{
								"id":    "b",
								"value": 1,
							},
							{
								"id":    "c",
								"value": 2,
							},
						},
						Code: `uniqByKeyLast $.input "value"`,
						Return: []map[string]any{
							{"id": "b", "value": 1},
							{"id": "c", "value": 2},
						},
					}, {
						Description: `Here we provide an expression that will normalize the value of the "name" field before performing the unique operation.`,
						Input: []map[string]any{
							{
								"name": "bob",
								"i":    1,
							},
							{
								"name": "BOB",
								"i":    2,
							},
							{
								"name": "Bob",
								"i":    3,
							},
						},
						Code: `uniqByKeyLast $.input "name" "{{ upper . }}"`,
						Return: []map[string]any{
							{"name": "Bob", "i": 3},
						},
					},
				},
				Function: func(input any, key string, exprs ...any) ([]any, error) {
					return uniqByKey(funcs, input, key, true, exprs...)
				},
			}, {

				Name:    `sortByKey`,
				Summary: `Sort the given array of objects by comparing the values of the given key for all objects.`,
				Arguments: []FuncArg{
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
				Examples: []FuncExample{
					{
						Input: []map[string]any{
							{
								"name": "bob",
							},
							{
								"name": "Mallory",
							},
							{
								"name": "ALICE",
							},
						},
						Code: `sortByKey $.input "name"`,
						Return: []map[string]any{
							{"name": "ALICE"},
							{"name": "Mallory"},
							{"name": "bob"},
						},
					},
				},
				Function: func(input any, key string) ([]any, error) {
					var out = sliceutil.Sliceify(input)
					sort.Slice(out, func(i int, j int) bool {
						var mI = maputil.M(out[i])
						var mJ = maputil.M(out[j])

						return typeutil.IsLessThan(
							mI.Get(key),
							mJ.Get(key),
						)
					})
					return out, nil
				},
			}, {

				Name:    `rSortByKey`,
				Summary: `Same as sortByKey, but reversed.`,
				Arguments: []FuncArg{
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
				Examples: []FuncExample{
					{
						Input: []map[string]any{
							{
								"name": "bob",
							},
							{
								"name": "Mallory",
							},
							{
								"name": "ALICE",
							},
						},
						Code: `rSortByKey $.input "name"`,
						Return: []map[string]any{
							{"name": "bob"},
							{"name": "Mallory"},
							{"name": "ALICE"},
						},
					},
				},
				Function: func(input any, key string) ([]any, error) {
					var out = sliceutil.Sliceify(input)
					sort.Slice(out, func(i int, j int) bool {
						var mI = maputil.M(out[i])
						var mJ = maputil.M(out[j])

						return typeutil.IsLessThan(
							mJ.Get(key),
							mI.Get(key),
						)
					})
					return out, nil
				},
			}, {
				Name:    `isortByKey`,
				Summary: `Sort the given array of objects by comparing the values of the given key for all objects (case-insensitive compare).`,
				Arguments: []FuncArg{
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
				Examples: []FuncExample{
					{
						Input: []map[string]any{
							{
								"name": "bob",
							},
							{
								"name": "Mallory",
							},
							{
								"name": "ALICE",
							},
						},
						Code: `isortByKey $.input "name"`,
						Return: []map[string]any{
							{"name": "ALICE"},
							{"name": "bob"},
							{"name": "Mallory"},
						},
					},
				},
				Function: func(input any, key string) ([]any, error) {
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
				Arguments: []FuncArg{
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
				Examples: []FuncExample{
					{
						Input: []map[string]any{
							{
								"name": "Bob",
							},
							{
								"name": "Mallory",
							},
							{
								"name": "Alice",
							},
						},
						Code: `irSortByKey $.input "name"`,
						Return: []map[string]any{
							{"name": "Mallory"},
							{"name": "Bob"},
							{"name": "Alice"},
						},
					},
				},
				Function: func(input any, key string) ([]any, error) {
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
				Arguments: []FuncArg{
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
				Examples: []FuncExample{
					{
						Input: []map[string]any{
							{
								"name": "Bob",
							},
							{
								"name": "Mallory",
							},
							{
								"name": "Alice",
							},
						},
						Code:   `pluck $.input "name"`,
						Return: []string{`Bob`, `Mallory`, `Alice`},
					},
				},
				Function: func(input any, key string, additionalKeys ...any) []any {
					var out = maputil.Pluck(input, strings.Split(key, `.`))

					for _, ak := range sliceutil.Stringify(additionalKeys) {
						out = append(out, maputil.Pluck(input, strings.Split(ak, `.`))...)
					}

					return out
				},
			}, {
				Name:    `keys`,
				Summary: `Return an array of key names specifying all the keys of the given object.`,
				Arguments: []FuncArg{
					{
						Name:        `object`,
						Type:        `object`,
						Description: `The object to return the key names from.`,
					},
				},
				Examples: []FuncExample{
					{
						Input: map[string]any{
							"id":    "a",
							"value": 1,
						},
						Code:   `keys $.input`,
						Return: []string{`id`, `value`},
					},
				},
				Function: func(input any) []any {
					var keys = maputil.Keys(input)

					sort.Slice(keys, func(i int, j int) bool {
						return typeutil.String(keys[i]) < typeutil.String(keys[j])
					})

					return keys
				},
			}, {
				Name:    `values`,
				Summary: `Return an array of values from the given object.`,
				Arguments: []FuncArg{
					{
						Name:        `object`,
						Type:        `object`,
						Description: `The object to return values from.`,
					},
				},
				Examples: []FuncExample{
					{
						Input: map[string]any{
							"id":    "a",
							"value": 1,
						},
						Code:   `values $.input`,
						Return: []any{1, `a`},
					},
				},
				Function: func(input any) []any {
					var values = maputil.MapValues(input)

					sort.Slice(values, func(i int, j int) bool {
						return typeutil.String(values[i]) < typeutil.String(values[j])
					})

					return values
				},
			}, {
				Name: `get`,
				Summary: `Retrieve a value from a given object.  Key can be specified as a dot.separated.list string or ` +
					`array of keys that describes a path from the given object, through any intermediate nested objects, ` +
					`down to the object containing the desired value.`,
				Arguments: []FuncArg{
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
				Examples: []FuncExample{
					{
						Input: map[string]any{
							"name": "Bob",
						},
						Code:   `get $.input "name"`,
						Return: `Bob`,
					},
					{
						Input: map[string]any{
							"properties": map[string]any{
								"info": map[string]any{
									"name": "Bob",
								},
							},
						},
						Code:   `get $.input "properties.info.name"`,
						Return: `Bob`,
					},
					{
						Input: map[string]any{
							"properties": map[string]any{
								"info": map[string]any{
									"name": "Bob",
								},
							},
						},
						Code:   `get $.input "properties.info.age"`,
						Return: ``,
					}, {
						Input: map[string]any{
							"properties": map[string]any{
								"info": map[string]any{
									"age": 42,
								},
							},
						},
						Code:   `get $.input "properties.info.age" 42`,
						Return: 42,
					},
				},
				Function: func(input any, key any, fallback ...any) any {
					var fb any

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
				Arguments: []FuncArg{
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
				Function: func(input any, key any, value any) error {
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
				Arguments: []FuncArg{
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
				Examples: []FuncExample{
					{
						Input: []map[string]any{
							{
								"id": 1,
								"children": []map[string]any{
									{"id": 3},
									{"id": 5},
									{"id": 8},
								},
							},
						},
						Code:   `findKey $.input "id"`,
						Return: []int{1, 3, 5, 8},
					},
				},
				Function: func(input any, key string) ([]any, error) {
					var values = make([]any, 0)

					if err := maputil.Walk(input, func(value any, path []string, isLeaf bool) error {
						if isLeaf && path[len(path)-1] == key {
							values = append(values, value)
						}

						return nil
					}); err != nil {
						return nil, err
					}

					sort.Slice(values, func(i int, j int) bool {
						return typeutil.String(values[i]) < typeutil.String(values[j])
					})

					return values, nil
				},
			}, {
				Name:    `has`,
				Summary: `Return whether a specific element is in an array.`,
				Arguments: []FuncArg{
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
				Examples: []FuncExample{
					{
						Input:  []string{"a", "e", "i", "o", "u"},
						Code:   `has "e" $.input`,
						Return: true,
					}, {
						Input:  []string{"a", "e", "i", "o", "u"},
						Code:   `has "y" $.input`,
						Return: false,
					}, {
						Input:  []string{"3", "5", "8", "13"},
						Code:   `has "13" $.input`,
						Return: true,
					}, {
						Input:  []string{"3", "5", "8", "13"},
						Code:   `has 13 $.input`,
						Return: true,
					}, {
						Input:  []string{"3", "5", "8", "13"},
						Code:   `has 14 $.input`,
						Return: false,
					},
				},
				Function: func(want any, input any) bool {
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
				Arguments: []FuncArg{
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
				Examples: []FuncExample{
					{
						Input:  []string{"a", "e", "i", "o", "u"},
						Code:   `any $.input "e" "y" "x"`,
						Return: true,
					},
					{
						Input:  []string{"r", "s", "t", "l", "n", "e"},
						Code:   `any $.input "f" "m" "w" "o"`,
						Return: false,
					},
				},
				Function: func(input any, wants ...any) bool {
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
				Arguments: []FuncArg{
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
				Examples: []FuncExample{
					{
						Input:  []string{"a", "e", "i", "o", "u"},
						Code:   `indexOf $.input "e"`,
						Return: 1,
					},
					{
						Input:  []string{"a", "e", "i", "o", "u"},
						Code:   `indexOf $.input "y"`,
						Return: -1,
					},
				},
				Function: func(slice any, value any) (index int) {
					index = -1

					if typeutil.IsArray(slice) {
						sliceutil.Each(slice, func(i int, v any) error {
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
				Arguments: []FuncArg{
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
				Examples: []FuncExample{
					{
						Input:  []string{`a`, `e`, `i`, `o`, `u`},
						Code:   `slice $.input 0 -1`,
						Return: []string{`a`, `e`, `i`, `o`, `u`},
					}, {
						Input:  []string{`a`, `e`, `i`, `o`, `u`},
						Code:   `slice $.input 2 -1`,
						Return: []string{`i`, `o`, `u`},
					}, {
						Input:  []string{`a`, `e`, `i`, `o`, `u`},
						Code:   `slice $.input -3 -1`,
						Return: []string{`i`, `o`, `u`},
					}, {
						Input:  []string{`a`, `e`, `i`, `o`, `u`},
						Code:   `slice $.input 1 2`,
						Return: []string{`e`},
					},
				},
				Function: func(slice any, from any, to any) []any {
					return sliceutil.Slice(
						slice,
						int(typeutil.Int(from)),
						int(typeutil.Int(to)),
					)
				},
			}, {
				Name:    `sslice`,
				Summary: `Identical to [slice](#fn-slice), but returns an array of strings.`,
				Function: func(slice any, from any, to any) []string {
					return sliceutil.StringSlice(slice, int(typeutil.Int(from)), int(typeutil.Int(to)))
				},
			}, {
				Name:    `uniq`,
				Summary: `Return an array containing only unique values from the given array.`,
				Arguments: []FuncArg{
					{
						Name:        `input`,
						Type:        `array`,
						Description: `The array to unique.`,
					},
				},
				Examples: []FuncExample{
					{
						Input:  []string{"a", "a", "b", "b", "b", "c"},
						Code:   `uniq $.input`,
						Return: []string{`a`, `b`, `c`},
					},
				},
				Function: func(slice any) []any {
					return sliceutil.Unique(slice)
				},
			}, {
				Name:    `flatten`,
				Summary: `Return an array of values with all nested arrays collapsed down to a single, flat array.`,
				Arguments: []FuncArg{
					{
						Name:        `input`,
						Type:        `array`,
						Description: `The array to flatten.`,
					},
				},
				Examples: []FuncExample{
					{
						Input:  []any{"a", []string{"a", "b"}, []any{"b", "b", []string{"c"}}},
						Code:   `flatten $.input`,
						Return: []string{`a`, `a`, `b`, `b`, `b`, `c`},
					},
				},
				Function: func(slice any) []any {
					return sliceutil.Flatten(slice)
				},
			}, {
				Name:    `compact`,
				Summary: `Return an copy of given array with all empty, null, and zero elements removed.`,
				Arguments: []FuncArg{
					{
						Name:        `input`,
						Type:        `array`,
						Description: `The array to compact.`,
					},
				},
				Examples: []FuncExample{
					{
						Input:  []any{"a", nil, "b", 0, false, "c"},
						Code:   `compact $.input`,
						Return: []any{`a`, `b`, 0, false, `c`},
					},
				},
				Function: func(slice any) []any {
					return sliceutil.Compact(slice)
				},
			}, {
				Name:    `first`,
				Summary: `Return the first value from the given array, or null if the array is empty.`,
				Arguments: []FuncArg{
					{
						Name:        `input`,
						Type:        `array`,
						Description: `The array to read from.`,
					},
				},
				Examples: []FuncExample{
					{
						Input:  []any{"a", "b", "c", "d"},
						Code:   `first $.input`,
						Return: `a`,
					},
				},
				Function: func(slice any) (out any, err error) {
					err = sliceutil.Each(slice, func(i int, value any) error {
						out = value
						return sliceutil.Stop
					})

					return
				},
			}, {
				Name: `rest`,
				Summary: `Return all but the first value from the given array, or an empty array of the given ` +
					`array's length is <= 1.`,
				Arguments: []FuncArg{
					{
						Name:        `input`,
						Type:        `array`,
						Description: `The array to read from.`,
					},
				},
				Examples: []FuncExample{
					{
						Input:  []any{"a", "b", "c", "d"},
						Code:   `rest $.input`,
						Return: []string{`b`, `c`, `d`},
					}, {
						Input:  []any{"a"},
						Code:   `rest $.input`,
						Return: []string{},
					},
				},
				Function: func(slice any) ([]any, error) {
					return sliceutil.Rest(slice), nil
				},
			}, {
				Name:    `last`,
				Summary: `Return the last value from the given array, or null if the array is empty.`,
				Arguments: []FuncArg{
					{
						Name:        `input`,
						Type:        `array`,
						Description: `The array to read from.`,
					},
				},
				Examples: []FuncExample{
					{
						Input:  []string{"a", "b", "c", "d"},
						Code:   `last $.input`,
						Return: `d`,
					},
				},
				Function: func(slice any) (out any, err error) {
					err = sliceutil.Each(slice, func(i int, value any) error {
						out = value
						return nil
					})

					return
				},
			}, {
				Name:    `count`,
				Summary: `Identical to the built-in "len" function, but is less picky about types.`,
				Arguments: []FuncArg{
					{
						Name:        `input`,
						Type:        `array`,
						Description: `The array to read from.`,
					},
				},
				Examples: []FuncExample{
					{
						Input:  []string{"a", "b", "c", "d"},
						Code:   `count $.input`,
						Return: 4,
					},
				},
				Function: func(in any) int {
					return sliceutil.Len(in)
				},
			}, {
				Name:    `sort`,
				Summary: `Return an array sorted in lexical ascending order.`,
				Arguments: []FuncArg{
					{
						Name:        `input`,
						Type:        `array`,
						Description: `The array to sort.`,
					},
				},
				Examples: []FuncExample{
					{
						Input:  []string{"d", "a", "c", "b"},
						Code:   `sort $.input`,
						Return: []string{`a`, `b`, `c`, `d`},
					},
				},
				Function: func(input any) []any {
					var out = sliceutil.Sliceify(input)

					sort.Slice(out, func(i, j int) bool {
						return typeutil.IsLessThan(
							out[i],
							out[j],
						)
					})

					return out
				},
			}, {
				Name:    `rsort`,
				Summary: `Return the array sorted in lexical descending order.`,
				Arguments: []FuncArg{
					{
						Name:        `input`,
						Type:        `array`,
						Description: `The array to sort.`,
					},
				},
				Examples: []FuncExample{
					{
						Input:  []string{"d", "a", "c", "b"},
						Code:   `rsort $.input`,
						Return: []string{`d`, `c`, `b`, `a`},
					},
				},
				Function: func(input any) []any {
					var out = sliceutil.Sliceify(input)

					sort.Slice(out, func(i, j int) bool {
						return typeutil.IsLessThan(
							out[j],
							out[i],
						)
					})

					return out
				},
			}, {
				Name:    `isort`,
				Summary: `Return an array sorted in lexical ascending order (case-insensitive.)`,
				Arguments: []FuncArg{
					{
						Name:        `input`,
						Type:        `array`,
						Description: `The array to sort.`,
					},
				},
				Examples: []FuncExample{
					{
						Input:  []string{"bob", "ALICE", "Mallory"},
						Code:   `isort $.input`,
						Return: []string{`ALICE`, `bob`, `Mallory`},
					},
				},
				Function: func(input any) []any {
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
				Arguments: []FuncArg{
					{
						Name:        `input`,
						Type:        `array`,
						Description: `The array to sort.`,
					},
				},
				Examples: []FuncExample{
					{
						Input:  []string{"bob", "ALICE", "Mallory"},
						Code:   `irsort $.input`,
						Return: []string{`Mallory`, `bob`, `ALICE`},
					},
				},
				Function: func(input any, keys ...string) []any {
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
				Arguments: []FuncArg{
					{
						Name:        `input`,
						Type:        `array`,
						Description: `The array to read from.`,
					},
				},
				Examples: []FuncExample{
					{
						Input:  []string{"a", "a", "b", "b", "b", "c"},
						Code:   `mostcommon $.input`,
						Return: `b`,
					},
				},
				Function: func(slice any) (any, error) {
					return commonses(slice, `most`)
				},
			}, {
				Name:    `leastcommon`,
				Summary: `Return the element in a given array that appears the least frequently.`,
				Arguments: []FuncArg{
					{
						Name:        `input`,
						Type:        `array`,
						Description: `The array to read from.`,
					},
				},
				Examples: []FuncExample{
					{
						Input:    []string{"a", "a", "b", "b", "b", "c"},
						Code:     `leastcommon $.input`,
						Return:   `c`,
						SkipTest: true,
					},
				},
				Function: func(slice any) (any, error) {
					return commonses(slice, `least`)
				},
			}, {
				Name: `sliceify`,
				Summary: `Convert the given input into an array.  If the value is already an array, ` +
					`this just returns that array.  Otherwise, it returns an array containing the ` +
					`given value as its only element.`,
				Arguments: []FuncArg{
					{
						Name:        `input`,
						Type:        `any`,
						Description: `The value to make into an array.`,
					},
				},
				Examples: []FuncExample{
					{
						Input:  []string{"a", "b", "c"},
						Code:   `sliceify $.input`,
						Return: []string{`a`, `b`, `c`},
					},
					{
						Code:   `sliceify 4`,
						Return: []int{4},
					},
				},
				Function: func(slice any) []any {
					return sliceutil.Sliceify(slice)
				},
			}, {
				Name: `stringify`,
				Summary: `Identical to [sliceify](#fn-sliceify), but converts all values to ` +
					`strings and returns an array of strings.`,
				Function: func(slice any) []string {
					return sliceutil.Stringify(slice)
				},
			}, {
				Name:    `intersect`,
				Summary: `Return the intersection of two arrays.`,
				Arguments: []FuncArg{
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
				Examples: []FuncExample{
					{
						Input: map[string]any{
							`first`:  []string{"b", "a", "c"},
							`second`: []string{"c", "b", "d"},
						},
						Code:   `intersect $.input.first $.input.second`,
						Return: []string{`b`, `c`},
					},
					{
						Input: map[string]any{
							`first`:  []string{"a", "b", "c"},
							`second`: []string{"x", "y", "z"},
						},
						Code:   `intersect $.input.first $.input.second`,
						Return: []string{},
					},
				},
				Function: func(first any, second any) []any {
					return sliceutil.Intersect(first, second)
				},
			}, {
				Name:    `difference`,
				Summary: `Return the first array with common elements from the second removed.`,
				Arguments: []FuncArg{
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
				Examples: []FuncExample{
					{
						Input: map[string]any{
							`first`:  []string{"b", "a", "c"},
							`second`: []string{"c", "b", "d"},
						},
						Code:   `difference $.input.first $.input.second`,
						Return: []string{`a`},
					},
					{
						Input: map[string]any{
							`first`:  []string{"a", "b", "c"},
							`second`: []string{"x", "y", "z"},
						},
						Code:   `difference $.input.first $.input.second`,
						Return: []string{`a`, `b`, `c`},
					},
				},
				Function: func(first any, second any) []any {
					return sliceutil.Difference(first, second)
				},
			}, {
				Name:    `mapify`,
				Summary: `Return the given value returned as a rangeable object.`,
				Function: func(input any) map[string]any {
					return maputil.DeepCopy(input)
				},
			}, {
				Name:    `onlyKeys`,
				Summary: `Return the given object with only the specified keys included.`,
				Arguments: []FuncArg{
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
				Examples: []FuncExample{
					{
						Input: map[string]any{
							`a`: 1,
							`b`: 2,
							`c`: 3,
						},
						Code: `onlyKeys $.input "a" "c"`,
						Return: map[string]any{
							`a`: 1,
							`c`: 3,
						},
					},
				},
				Function: func(input any, keys ...string) map[string]any {
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
				Arguments: []FuncArg{
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
				Examples: []FuncExample{
					{
						Input: map[string]any{
							`a`: 1,
							`b`: 2,
							`c`: 3,
						},
						Code: `exceptKeys $.input "a" "c"`,
						Return: map[string]any{
							`b`: 2,
						},
					},
				},
				Function: func(input any, keys ...any) map[string]any {
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
				Arguments: []FuncArg{
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
				Examples: []FuncExample{
					{
						Input: []map[string]any{
							{"name": "Bob", "title": "Friend"},
							{"name": "Mallory", "title": "Foe"},
							{"name": "Alice", "title": "Friend"},
						},
						Code: `groupBy $.input "title"`,
						Return: map[string][]any{
							`Friend`: {
								map[string]any{
									`name`:  `Bob`,
									`title`: `Friend`,
								},
								map[string]any{
									`name`:  `Alice`,
									`title`: `Friend`,
								},
							},
							`Foe`: {
								map[string]any{
									`name`:  `Mallory`,
									`title`: `Foe`,
								},
							},
						},
					},
				},
				Function: func(sliceOfMaps any, key string, tpls ...any) (map[string][]any, error) {
					if !typeutil.IsArray(sliceOfMaps) {
						return nil, fmt.Errorf("groupBy only works on arrays of objects, got %T", sliceOfMaps)
					}

					var output = make(map[string][]any)
					var valueTpls = sliceutil.Stringify(tpls)

					if items := sliceutil.Sliceify(sliceOfMaps); len(items) > 0 {
						if !typeutil.IsMap(items[0]) {
							return nil, fmt.Errorf("groupBy only works on arrays of objects, got %T", items[0])
						}

						for _, item := range items {
							var value = maputil.DeepGet(item, strings.Split(key, `.`))

							if len(valueTpls) > 0 && valueTpls[0] != `` {
								if stringutil.IsSurroundedBy(valueTpls[0], `{{`, `}}`) {
									var tmpl = NewTemplateWithFuncs(`inline`, TextEngine, funcs)

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
								output[valueS] = []any{item}
							}
						}
					}

					return output, nil
				},
			}, {
				Name:    `head`,
				Summary: `Return the first _n_ items from an array.`,
				Arguments: []FuncArg{
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
				Examples: []FuncExample{
					{
						Input:  []string{"a", "b", "c", "d"},
						Code:   `head $.input 2`,
						Return: []string{`a`, `b`},
					},
				},
				Function: func(input any, n int) []any {
					if typeutil.IsZero(input) {
						return make([]any, 0)
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
				Arguments: []FuncArg{
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
				Examples: []FuncExample{
					{
						Input:  []string{"a", "b", "c", "d"},
						Code:   `tail $.input 2`,
						Return: []string{`c`, `d`},
					},
				},
				Function: func(input any, n int) []any {
					if typeutil.IsZero(input) {
						return make([]any, 0)
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
				Arguments: []FuncArg{
					{
						Name:        `input`,
						Type:        `array`,
						Description: `The array to shuffle.`,
					},
				},
				Examples: []FuncExample{
					{
						Input:    []string{"a", "b", "c", "d"},
						Code:     `shuffle $.input`,
						Return:   []string{`d`, `c`, `b`, `a`},
						SkipTest: true,
					},
				},
				Function: func(input ...any) []any {
					if typeutil.IsZero(input) {
						return make([]any, 0)
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
				Arguments: []FuncArg{
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
				Function: func(input any, seeds ...int64) (int64, error) {
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
				Arguments: []FuncArg{
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
				Examples: []FuncExample{
					{
						Input:  []string{"a", "B", "C", "d"},
						Code:   `apply $.input "upper"`,
						Return: []string{`A`, `B`, `C`, `D`},
					},
					{
						Input:  []string{"a", "B", "C", "d"},
						Code:   `apply $.input "upper" "lower"`,
						Return: []string{`a`, `b`, `c`, `d`},
					},
				},
				Function: func(input any, fns ...string) ([]any, error) {
					var out = make([]any, 0)

					if err := sliceutil.Each(input, func(i int, value any) error {
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
				Arguments: []FuncArg{
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
				Examples: []FuncExample{
					{
						Input: map[string]any{
							"properties/enabled": true,
							"properties/label":   "items",
							"name":               "Items",
							"properties/tasks/0": "do things",
							"properties/tasks/1": "do stuff",
						},
						Code: `diffuse $.input "/"`,
						Return: map[string]any{
							`name`: `Items`,
							`properties`: map[string]any{
								`enabled`: true,
								`label`:   `items`,
								`tasks`: []any{
									`do things`,
									`do stuff`,
								},
							},
						},
					},
				},
				Function: func(input any, joiner string) (map[string]any, error) {
					if in, err := prepCoalesceDiffuseInput(input); err == nil {
						return maputil.DiffuseMap(in, joiner)
					} else {
						return nil, fmt.Errorf("can only diffuse arrays and objects, got %T", input)
					}
				},
			}, {
				Name: `coalesce`,
				Summary: `Convert an array of objects or object representing a deeply-nested ` +
					`hierarchy of items and collapse it into a flat (not nested) object.`,
				Arguments: []FuncArg{
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
				Function: func(input any, joiner string) (map[string]any, error) {
					if in, err := prepCoalesceDiffuseInput(input); err == nil {
						return maputil.CoalesceMap(in, joiner)
					} else {
						return nil, fmt.Errorf("can only coalesce arrays and objects, got %T", input)
					}
				},
			}, {
				Name:    `isLastElement`,
				Summary: `Returns whether the given index in the given array is the last element in that array.`,
				Arguments: []FuncArg{
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
				Function: func(index any, array any) bool {
					var i = typeutil.Int(index)
					var arr = sliceutil.Sliceify(array)

					return (int(i) == (len(arr) - 1))
				},
			}, {
				Name:    `jsonPath`,
				Summary: `Returns the input object filtered using the given JSONPath query.`,
				Arguments: []FuncArg{
					{
						Name:        `query`,
						Type:        `string`,
						Description: `The JSONPath query to filter by.`,
					}, {
						Name:        `data`,
						Type:        `object`,
						Description: `The object being filtered.`,
					},
				},
				Function: func(query string, data any) (any, error) {
					return maputil.JSONPath(data, query)
				},
			}, {
				Name:    `objectify`,
				Summary: ``,
				Arguments: []FuncArg{
					{
						Name:        `input`,
						Type:        `array`,
						Description: `An array of objects or strings to parse and convert into an object`,
					}, {
						Name:        `keyField`,
						Type:        `string`,
						Description: `The name of the field in the parsed input whose value will become the keys in the resultant object.`,
					}, {
						Name:        `valueField`,
						Type:        `string`,
						Description: `The name of the field in the parsed input whose value will become the values in the resultant object.`,
					}, {
						Name:        `keyValueSeparator`,
						Optional:    true,
						Type:        `string`,
						Description: `If given an array of strings, the first occurrence of this string will separate the key and value portions of the string.`,
					},
				},
				Examples: []FuncExample{
					{
						Input: []map[string]any{
							{
								"label": "First Name",
								"value": "firstName",
							},
							{
								"label": "Last Name",
								"value": "lastName",
							},
						},
						Code: `objectify $.input "value" "label"`,
						Return: map[string]any{
							`firstName`: `First Name`,
							`lastName`:  `Last Name`,
						},
					},
				},
				Function: func(input any, keyField string, valueField string, kvs ...string) map[string]any {
					var result = make(map[string]any)
					var keyValueSeparator = typeutil.String(
						sliceutil.FirstNonZero(kvs, DefaultObjectifyKeyValueSeparator),
					)

					if typeutil.IsMap(input) {
						var m = maputil.M(input)

						if key := m.String(keyField); key != `` {
							if valueField != `` {
								result[key] = m.Get(valueField).Value
							} else {
								result[key] = nil
							}
						}
					} else {
						var items = sliceutil.Sliceify(input)

						for _, item := range items {
							var key string
							var value any

							if typeutil.IsMap(item) {
								if keyField != `` {
									var m = maputil.M(item)

									if k := m.String(keyField); k != `` {
										key = k

										if valueField != `` {
											value = m.Get(valueField).Value
										}
									}
								}
							} else {
								key, value = stringutil.SplitPairTrimSpaceAuto(
									typeutil.String(item),
									keyValueSeparator,
								)
							}

							if key != `` {
								result[key] = value
							}
						}
					}

					return result
				},
			},
		},
	}

	group.Functions = append(group.Functions, []FuncDef{
		{
			Name:     `findkey`,
			Alias:    `findKey`,
			Function: group.fn(`findKey`),
			Hidden:   true,
		},
	}...)

	return group
}

func prepCoalesceDiffuseInput(input any) (map[string]any, error) {
	var in = make(map[string]any)

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
