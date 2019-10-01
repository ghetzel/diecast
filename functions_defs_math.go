package diecast

import (
	"fmt"
	"math"

	"github.com/ghetzel/go-stockutil/mathutil"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/typeutil"
	"github.com/montanaflynn/stats"
)

func loadStandardFunctionsMath(funcs FuncMap, server *Server) funcGroup {
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
			}, {
				Name:    `isEven`,
				Summary: `Return whether the given number is even.`,
				Function: func(number interface{}) bool {
					return (math.Mod(typeutil.Float(number), 2) == 0)
				},
			}, {
				Name:    `isOdd`,
				Summary: `Return whether the given number is odd.`,
				Function: func(number interface{}) bool {
					return (math.Mod(typeutil.Float(number), 2) != 0)
				},
			}, {
				Name:    `abs`,
				Summary: `Return the absolute value of the given number.`,
				Function: func(number interface{}) float64 {
					return math.Abs(typeutil.Float(number))
				},
			}, {
				Name:    `ceil`,
				Summary: `Return the greatest integer value greater than or equal to the given number.`,
				Function: func(number interface{}) float64 {
					return math.Ceil(typeutil.Float(number))
				},
			}, {
				Name:    `floor`,
				Summary: `Return the greatest integer value less than or equal to the given number.`,
				Function: func(number interface{}) float64 {
					return math.Floor(typeutil.Float(number))
				},
			}, {
				Name:    `sin`,
				Summary: `Return the sine of the given number (in radians).`,
				Function: func(rad interface{}) float64 {
					return math.Sin(typeutil.Float(rad))
				},
			}, {
				Name:    `cos`,
				Summary: `Return the cosine of the given number (in radians).`,
				Function: func(rad interface{}) float64 {
					return math.Cos(typeutil.Float(rad))
				},
			}, {
				Name:    `tan`,
				Summary: `Return the tangent of the given number (in radians).`,
				Function: func(rad interface{}) float64 {
					return math.Tan(typeutil.Float(rad))
				},
			}, {
				Name:    `asin`,
				Summary: `Return the arcsine of the given number (in radians).`,
				Function: func(rad interface{}) float64 {
					return math.Asin(typeutil.Float(rad))
				},
			}, {
				Name:    `acos`,
				Summary: `Return the arccosine of the given number (in radians).`,
				Function: func(rad interface{}) float64 {
					return math.Acos(typeutil.Float(rad))
				},
			}, {
				Name:    `atan`,
				Summary: `Return the arctangent of the given number (in radians).`,
				Function: func(rad interface{}) float64 {
					return math.Atan(typeutil.Float(rad))
				},
			}, {
				Name:    `deg2rad`,
				Summary: `Return the given number of degrees in radians.`,
				Function: func(deg interface{}) float64 {
					return typeutil.Float(deg) * (math.Pi / 180)
				},
			}, {
				Name:    `rad2deg`,
				Summary: `Return the given number of radians in degrees.`,
				Function: func(rad interface{}) float64 {
					return typeutil.Float(rad) * (180 / math.Pi)
				},
			},
		},
	}

	// Numeric Aggregation Functions
	// ---------------------------------------------------------------------------------------------
	for _, obj := range []statsUnary{
		{
			Name:     `maximum`,
			Function: stats.Max,
		}, {
			Name:     `mean`,
			Function: stats.Mean,
		}, {
			Name:     `median`,
			Function: stats.Median,
		}, {
			Name:     `minimum`,
			Function: stats.Min,
		}, {
			Name:     `minimum_nz`,
			Function: MinNonZero,
		}, {
			Name:     `stddev`,
			Function: stats.StandardDeviation,
		}, {
			Name:     `sum`,
			Function: stats.Sum,
		},
	} {
		docName := obj.Name

		switch docName {
		case `minimum_nz`:
			docName = `minimum (excluding zero)`
		case `stddev`:
			docName = `standard deviation`
		}

		group.Functions = append(group.Functions, funcDef{
			Name:    obj.Name,
			Summary: fmt.Sprintf("Return the %s of the given array of numbers.", docName),
			Function: func(statsFn statsUnaryFn) statsTplFunc {
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
			}(obj.Function),
		})
	}

	return group
}
