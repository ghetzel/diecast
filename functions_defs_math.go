package diecast

import (
	"fmt"
	"github.com/ghetzel/go-stockutil/mathutil"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/typeutil"
	"github.com/montanaflynn/stats"
)

func loadStandardFunctionsMath(funcs FuncMap) funcGroup {
	group := funcGroup{
		Name:        `Math and Statistics`,
		Description: `These functions implement basic mathematical and statistical operations on numbers.`,
		Functions: []funcDef{
			{
				Name:     `calc`,
				Summary:  ``,
				Function: calcFn,
			}, {
				Name:    `add`,
				Summary: `Return the sum of all of the given values.`,
				Function: func(values ...interface{}) float64 {
					out, _ := calcFn(`+`, values...)
					return out
				},
			}, {
				Name:    `subtract`,
				Summary: `Sequentially subtract all of the given values.`,
				Function: func(values ...interface{}) float64 {
					out, _ := calcFn(`-`, values...)
					return out
				},
			}, {
				Name:    `multiply`,
				Summary: `Return the product of all of the given values.`,
				Function: func(values ...interface{}) float64 {
					out, _ := calcFn(`*`, values...)
					return out
				},
			}, {
				Name:    `divide`,
				Summary: `Sequentially divide all of the given values in the order given.`,
				Function: func(values ...interface{}) (float64, error) {
					return calcFn(`/`, values...)
				},
			}, {
				Name:    `mod`,
				Summary: `Return the modulus of all of the given values.`,
				Function: func(values ...interface{}) (float64, error) {
					return calcFn(`%`, values...)
				},
			}, {
				Name:    `pow`,
				Summary: `Sequentially exponentiate of all of the given *values*.`,
				Function: func(values ...interface{}) (float64, error) {
					return calcFn(`^`, values...)
				},
			}, {
				Name:    `sequence`,
				Summary: `Return an array of integers representing a sequence from [0, _n_).`,
				Function: func(max interface{}) []int {
					if v, err := stringutil.ConvertToInteger(max); err == nil {
						seq := make([]int, v)

						for i, _ := range seq {
							seq[i] = i
						}

						return seq
					} else {
						return nil
					}
				},
			}, {
				Name:    `round`,
				Summary: `Round a number to the nearest _n_ places.`,
				Function: func(in interface{}, places ...int) (float64, error) {
					if inF, err := stringutil.ConvertToFloat(in); err == nil {
						n := 0

						if len(places) > 0 {
							n = places[0]
						}

						if n > 0 {
							return mathutil.RoundPlaces(inF, n), nil
						} else {
							return mathutil.Round(inF), nil
						}
					} else {
						return 0, err
					}
				},
			}, {
				Name:    `negate`,
				Summary: `Return the given number multiplied by -1.`,
				Function: func(value interface{}) float64 {
					return -1 * typeutil.V(value).Float()
				},
			},
		},
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
		docName := fnName

		switch docName {
		case `minimum_nz`:
			docName = `minimum (excluding zero)`
		case `stddev`:
			docName = `standard deviation`
		}

		group.Functions = append(group.Functions, funcDef{
			Name:    fnName,
			Summary: fmt.Sprintf("Return the %s of the given array of numbers.", docName),
			Function: func(statsFn statsUnary) statsTplFunc {
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
			}(fn),
		})
	}

	return group
}
