package diecast

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/typeutil"
)

func loadStandardFunctionsCollections(rv FuncMap) {
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

	// fn groupBy: Return the given *input* array-of-objects as an object, keyed on the value of the
	//             specified group *field*.  The field argument can be a template.
	rv[`groupBy`] = func(sliceOfMaps interface{}, key string, valueTpls ...string) (map[string][]interface{}, error) {
		if !typeutil.IsArray(sliceOfMaps) {
			return nil, fmt.Errorf("groupBy only works on arrays of objects, got %T", sliceOfMaps)
		}

		output := make(map[string][]interface{})

		if items := sliceutil.Sliceify(sliceOfMaps); len(items) > 0 {
			if !typeutil.IsMap(items[0]) {
				return nil, fmt.Errorf("groupBy only works on arrays of objects, got %T", items[0])
			}

			for _, item := range items {
				value := maputil.DeepGet(item, strings.Split(key, `.`))

				if len(valueTpls) > 0 && valueTpls[0] != `` {
					if stringutil.IsSurroundedBy(valueTpls[0], `{{`, `}}`) {
						tmpl := NewTemplate(`inline`, TextEngine)
						tmpl.Funcs(rv)

						if err := tmpl.Parse(valueTpls[0]); err == nil {
							output := bytes.NewBuffer(nil)

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

				valueS := fmt.Sprintf("%v", value)

				if v, ok := output[valueS]; ok {
					output[valueS] = append(v, item)
				} else {
					output[valueS] = []interface{}{item}
				}
			}
		}

		return output, nil
	}
}
