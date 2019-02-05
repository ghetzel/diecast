package diecast

import (
	"bytes"
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"strings"

	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/typeutil"
)

var errorInterface = reflect.TypeOf((*error)(nil)).Elem()

func loadStandardFunctionsCollections(funcs FuncMap) funcGroup {
	return funcGroup{
		Name: `Arrays and Objects`,
		Description: `For converting, modifying, and filtering arrays, objects, and arrays of ` +
			`objects. These functions are especially useful when working with data returned from Bindings.`,
		Functions: []funcDef{
			{
				Name: `page`,
				Summary: `Returns an integer representing an offset used for accessing paginated values when ` +
					`given a page number and number of results per page.`,
				Function: func(pagenum interface{}, perpage interface{}) int {
					factor := typeutil.V(pagenum).Int() - 1
					per := typeutil.V(perpage).Int()

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
				Function: func(input interface{}) []interface{} {
					array := sliceutil.Sliceify(input)
					output := make([]interface{}, len(array))

					for i := 0; i < len(array); i++ {
						output[len(array)-1-i] = array[i]
					}

					return output
				},
			}, {
				Name:    `filter`,
				Summary: `Return the given array with only elements where expression evaluates to a truthy value.`,
				Function: func(input interface{}, expr string) ([]interface{}, error) {
					out := make([]interface{}, 0)

					for i, value := range sliceutil.Sliceify(input) {
						tmpl := NewTemplate(`inline`, TextEngine)
						tmpl.Funcs(funcs)

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
				},
			}, {
				Name: `filterByKey`,
				Summary: `Return a subset of the elements in the given array whose values are objects ` +
					`that contain the given key.  Optionally, the values at the key for each object in ` +
					`the array can be passed to a template expression.  If that expression produces a ` +
					`truthy value, the object will be included in the output.  Otherwise it will not.`,
				Function: func(input interface{}, key string, exprs ...interface{}) ([]interface{}, error) {
					return filterByKey(funcs, input, key, exprs...)
				},
				Examples: []funcExample{},
			}, {
				Name: `firstByKey`,
				Summary: `Identical to [filterByKey](#fn-filterByKey), except it returns only the first ` +
					`object in the resulting array instead of the whole array.`,
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
				Function: func(input interface{}, key string, expr string) ([]interface{}, error) {
					out := make([]interface{}, 0)

					for i, obj := range sliceutil.Sliceify(input) {
						tmpl := NewTemplate(`inline`, TextEngine)
						tmpl.Funcs(funcs)
						m := maputil.M(obj)

						if !strings.HasPrefix(expr, `{{`) {
							expr = `{{` + expr
						}

						if !strings.HasSuffix(expr, `}}`) {
							expr = expr + `}}`
						}

						if err := tmpl.Parse(expr); err == nil {
							output := bytes.NewBuffer(nil)
							value := m.Auto(key)

							if err := tmpl.Render(output, value, ``); err == nil {
								evalValue := stringutil.Autotype(output.String())
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
				Function: func(input interface{}, key string, exprs ...interface{}) ([]interface{}, error) {
					return uniqByKey(funcs, input, key, false, exprs...)
				},
			}, {
				Name: `uniqByKeyLast`,
				Summary: `Identical to [uniqByKey](#fn-uniqByKey), except the _last_ of a set of objects grouped ` +
					`by key is included in the output, not the first.`,
				Function: func(input interface{}, key string, exprs ...interface{}) ([]interface{}, error) {
					return uniqByKey(funcs, input, key, true, exprs...)
				},
			}, {

				Name:    `sortByKey`,
				Summary: `Sort the given array of objects by comparing the values of the given key for all objects.`,
				Function: func(input interface{}, key string) ([]interface{}, error) {
					out := sliceutil.Sliceify(input)
					sort.Slice(out, func(i int, j int) bool {
						mI := maputil.M(out[i])
						mJ := maputil.M(out[j])
						return mI.String(key) < mJ.String(key)
					})
					return out, nil
				},
			}, {
				Name:    `pluck`,
				Summary: `Retrieve a value at the given key from each object in a given array of objects.`,
				Function: func(input interface{}, key string, additionalKeys ...string) []interface{} {
					out := maputil.Pluck(input, strings.Split(key, `.`))

					for _, ak := range additionalKeys {
						out = append(out, maputil.Pluck(input, strings.Split(ak, `.`))...)
					}

					return out
				},
			}, {
				Name:    `keys`,
				Summary: `Return an array of key names specifying all the keys of the given object.`,
				Function: func(input interface{}) []interface{} {
					return maputil.Keys(input)
				},
			}, {
				Name:    `values`,
				Summary: `Return an array of values from the given object.`,
				Function: func(input interface{}) []interface{} {
					return maputil.MapValues(input)
				},
			}, {
				Name: `get`,
				Summary: `Retrieve a value from a given object.  Key can be specified as a dot.separated.list of ` +
					`keys that describes a path from the given object, through any intermediate nested objects, ` +
					`down to the object containing the desired value.`,
				Function: func(input interface{}, key string, fallback ...interface{}) interface{} {
					var fb interface{}

					if len(fallback) > 0 {
						fb = fallback[0]
					}

					return maputil.DeepGet(input, strings.Split(key, `.`), fb)
				},
			}, {
				Name:    `findkey`,
				Summary: `Recursively scans the given array or map and returns all values of the given key.`,
				Function: func(input interface{}, key string) ([]interface{}, error) {
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
				},
			}, {
				Name:    `has`,
				Summary: `Return whether a specific element is in an array.`,
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
				Summary: `Return whether an array contains any of a set of desired.`,
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
				Summary: `Return a subset of the given array.`,
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
				Function: func(slice interface{}) []interface{} {
					return sliceutil.Unique(slice)
				},
			}, {
				Name:    `flatten`,
				Summary: `Return an array of values with all nested arrays collapsed down to a single, flat array.`,
				Function: func(slice interface{}) []interface{} {
					return sliceutil.Flatten(slice)
				},
			}, {
				Name:    `compact`,
				Summary: `Return an copy of given array with all empty, null, and zero elements removed.`,
				Function: func(slice interface{}) []interface{} {
					return sliceutil.Compact(slice)
				},
			}, {
				Name:    `first`,
				Summary: `Return the first value from the given array, or null if the array is empty.`,
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
				Function: func(slice interface{}) ([]interface{}, error) {
					return sliceutil.Rest(slice), nil
				},
			}, {
				Name:    `last`,
				Summary: `Return the last value from the given array, or null if the array is empty.`,
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
				Function: func(in interface{}) int {
					return sliceutil.Len(in)
				},
			}, {
				Name:    `sort`,
				Summary: `Return an array sorted in lexical ascending order.`,
				Function: func(input interface{}, keys ...string) []interface{} {
					return sorter(input, false, keys...)
				},
			}, {
				Name:    `rsort`,
				Summary: `Return the array sorted in lexical descending order.`,
				Function: func(input interface{}, keys ...string) []interface{} {
					return sorter(input, true, keys...)
				},
			}, {
				Name:    `isort`,
				Summary: `Return an array sorted in lexical ascending order (case-insensitive.)`,
				Function: func(input interface{}, keys ...string) []interface{} {
					return sorter(sliceutil.MapString(input, func(_ int, v string) string {
						return strings.ToLower(v)
					}), true, keys...)
				},
			}, {
				Name:    `irsort`,
				Summary: `Return the array sorted in lexical descending order (case-insensitive.)`,
				Function: func(input interface{}, keys ...string) []interface{} {
					return sorter(sliceutil.MapString(input, func(_ int, v string) string {
						return strings.ToLower(v)
					}), true, keys...)
				},
			}, {
				Name:    `mostcommon`,
				Summary: `Return the element in a given array that appears the most frequently.`,
				Function: func(slice interface{}) (interface{}, error) {
					return commonses(slice, `most`)
				},
			}, {
				Name:    `leastcommon`,
				Summary: `Return the element in a given array that appears the least frequently.`,
				Function: func(slice interface{}) (interface{}, error) {
					return commonses(slice, `least`)
				},
			}, {
				Name: `sliceify`,
				Summary: `Convert the given input into an array.  If the value is already an array, ` +
					`this just returns that array.  Otherwise, it returns an array containing the ` +
					`given value as its only element.`,
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
				Function: func(first interface{}, second interface{}) []interface{} {
					return sliceutil.Intersect(first, second)
				},
			}, {
				Name:    `mapify`,
				Summary: `Return the given value returned as a rangeable object.`,
				Function: func(input interface{}) map[string]interface{} {
					return maputil.DeepCopy(input)
				},
			}, {

				Name: `groupBy`,
				Summary: `Return the given array of objects as a grouped object, keyed on the ` +
					`value of the specified group field. The field argument can be an ` +
					`expression that receives the value and returns a transformed version of it.`,
				Function: func(sliceOfMaps interface{}, key string, valueTpls ...string) (map[string][]interface{}, error) {
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
									tmpl.Funcs(funcs)

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
				},
			}, {
				Name:    `head`,
				Summary: `Return the first _n_ items from an array.`,
				Function: func(input interface{}, n int) []interface{} {
					if typeutil.IsZero(input) {
						return make([]interface{}, 0)
					}

					items := sliceutil.Sliceify(input)

					if len(items) < n {
						return items
					} else {
						return items[0:n]
					}
				},
			}, {
				Name:    `tail`,
				Summary: `Return the last _n_ items from an array.`,
				Function: func(input interface{}, n int) []interface{} {
					if typeutil.IsZero(input) {
						return make([]interface{}, 0)
					}

					items := sliceutil.Sliceify(input)

					if len(items) < n {
						return items
					} else {
						return items[len(items)-n:]
					}
				},
			}, {
				Name:    `shuffle`,
				Summary: `Return the array with the elements rearranged in random order.`,
				Function: func(input ...interface{}) []interface{} {
					if typeutil.IsZero(input) {
						return make([]interface{}, 0)
					}

					inputS := sliceutil.Sliceify(input)

					for i := range inputS {
						j := rand.Intn(i + 1)
						inputS[i], inputS[j] = inputS[j], inputS[i]
					}

					return inputS
				},
			}, {
				Name: `apply`,
				Summary: `Apply a function to each of the elements in the given array. Note ` +
					`that functions must be unary (accept one argument of type _any_).`,
				Function: func(input interface{}, fns ...string) ([]interface{}, error) {
					out := make([]interface{}, 0)

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
			},
		},
	}
}
