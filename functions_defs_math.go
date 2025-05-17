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

func loadStandardFunctionsMath(_ FuncMap, _ *Server) funcGroup {
	var group = funcGroup{
		Name:        `Math and Statistics`,
		Description: `These functions implement basic mathematical and statistical operations on numbers.`,
		Functions: []funcDef{
			{
				Name:    `calc`,
				Summary: `Perform arbitrary calculations on zero or more numbers.`,
				Arguments: []funcArg{
					{
						Name:        `operator`,
						Type:        `string`,
						Description: `An operation to perform on the given sequence of numbers.`,
						Valid: []funcArg{

							{
								Name:        `+`,
								Description: `Addition operator`,
							}, {
								Name:        `-`,
								Description: `Subtraction operator`,
							}, {
								Name:        `*`,
								Description: `Multiply operator`,
							}, {
								Name:        `/`,
								Description: `Division operator`,
							}, {
								Name:        `^`,
								Description: `Exponent operator`,
							}, {
								Name:        `%`,
								Description: `Modulus operator`,
							},
						},
					}, {
						Name:        `numbers`,
						Type:        `float, integer`,
						Description: `Zero or more numbers to perform the given operation on (in order).`,
						Variadic:    true,
					},
				},
				Function: calcFn,
			}, {
				Name:    `add`,
				Summary: `Return the sum of all of the given values.`,
				Arguments: []funcArg{
					{
						Name:        `number`,
						Type:        `float, integer`,
						Description: `The number(s) to add together.`,
						Variadic:    true,
					},
				},
				Function: func(values ...any) float64 {
					out, _ := calcFn(`+`, values...)
					return out
				},
			}, {
				Name:    `subtract`,
				Summary: `Sequentially subtract all of the given values.`,
				Arguments: []funcArg{
					{
						Name:        `number`,
						Type:        `float, integer`,
						Description: `The number(s) to subtract, in order.`,
						Variadic:    true,
					},
				},
				Function: func(values ...any) float64 {
					out, _ := calcFn(`-`, values...)
					return out
				},
			}, {
				Name:    `multiply`,
				Summary: `Return the product of all of the given values.`,
				Arguments: []funcArg{
					{
						Name:        `number`,
						Type:        `float, integer`,
						Description: `The number(s) to multiply together.`,
						Variadic:    true,
					},
				},
				Function: func(values ...any) float64 {
					out, _ := calcFn(`*`, values...)
					return out
				},
			}, {
				Name:    `divide`,
				Summary: `Sequentially divide all of the given values in the order given.`,
				Arguments: []funcArg{
					{
						Name:        `number`,
						Type:        `float, integer`,
						Description: `The number(s) to divide, in order.`,
						Variadic:    true,
					},
				},
				Function: func(values ...any) (float64, error) {
					return calcFn(`/`, values...)
				},
			}, {
				Name:    `mod`,
				Summary: `Return the modulus of all of the given values.`,
				Arguments: []funcArg{
					{
						Name:        `number`,
						Type:        `float, integer`,
						Description: `The number(s) to operate on, in order.`,
						Variadic:    true,
					},
				},
				Function: func(values ...any) (float64, error) {
					return calcFn(`%`, values...)
				},
			}, {
				Name:    `pow`,
				Summary: `Sequentially exponentiate of all of the given *values*.`,
				Arguments: []funcArg{
					{
						Name:        `number`,
						Type:        `float, integer`,
						Description: `The number(s) to operate on, in order.`,
						Variadic:    true,
					},
				},
				Function: func(values ...any) (float64, error) {
					return calcFn(`^`, values...)
				},
			}, {
				Name:    `sequence`,
				Summary: `Return an array of integers representing a sequence from [0, _n_).`,
				Arguments: []funcArg{
					{
						Name:        `end`,
						Type:        `integer`,
						Description: `The largest number in the sequence (exclusive).`,
					},
					{
						Name:        `start`,
						Type:        `integer`,
						Description: `The number to start the sequence at (inclusive)`,
						Optional:    true,
						Default:     0,
					},
				},
				Function: func(max any, starts ...any) []int {
					var start = 0

					if len(starts) > 0 {
						start = int(typeutil.Int(starts[0]))
					}

					if v, err := stringutil.ConvertToInteger(max); err == nil {
						var seq = make([]int, v)

						for i := range seq {
							seq[i] = start + i
						}

						return seq
					} else {
						return nil
					}
				},
			}, {
				Name:    `round`,
				Summary: `Round a number to the nearest _n_ places.`,
				Arguments: []funcArg{
					{
						Name:        `number`,
						Type:        `float, integer`,
						Description: `The number to round.`,
					},
				},
				Function: func(in any, places ...int) (float64, error) {
					if inF, err := stringutil.ConvertToFloat(in); err == nil {
						var n = 0

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
				Arguments: []funcArg{
					{
						Name:        `number`,
						Type:        `float, integer`,
						Description: `The number to operate on.`,
					},
				},
				Function: func(value any) float64 {
					return -1 * typeutil.V(value).Float()
				},
			}, {
				Name:    `isEven`,
				Summary: `Return whether the given number is even.`,
				Arguments: []funcArg{
					{
						Name:        `number`,
						Type:        `float, integer`,
						Description: `The number to test.`,
					},
				},
				Function: func(number any) bool {
					return (math.Mod(typeutil.Float(number), 2) == 0)
				},
			}, {
				Name:    `isOdd`,
				Summary: `Return whether the given number is odd.`,
				Arguments: []funcArg{
					{
						Name:        `number`,
						Type:        `float, integer`,
						Description: `The number to test.`,
					},
				},
				Function: func(number any) bool {
					return (math.Mod(typeutil.Float(number), 2) != 0)
				},
			}, {
				Name:    `abs`,
				Summary: `Return the absolute value of the given number.`,
				Arguments: []funcArg{
					{
						Name:        `number`,
						Type:        `float, integer`,
						Description: `The number to operate on.`,
					},
				},
				Function: func(number any) float64 {
					return math.Abs(typeutil.Float(number))
				},
			}, {
				Name:    `ceil`,
				Summary: `Return the greatest integer value greater than or equal to the given number.`,
				Arguments: []funcArg{
					{
						Name:        `number`,
						Type:        `float, integer`,
						Description: `The number to operate on.`,
					},
				},
				Function: func(number any) float64 {
					return math.Ceil(typeutil.Float(number))
				},
			}, {
				Name:    `floor`,
				Summary: `Return the greatest integer value less than or equal to the given number.`,
				Arguments: []funcArg{
					{
						Name:        `number`,
						Type:        `float, integer`,
						Description: `The number to operate on.`,
					},
				},
				Function: func(number any) float64 {
					return math.Floor(typeutil.Float(number))
				},
			}, {
				Name:    `sin`,
				Summary: `Return the sine of the given number (in radians).`,
				Arguments: []funcArg{
					{
						Name:        `radians`,
						Type:        `float, integer`,
						Description: `The value (in radians) to operate on.`,
					},
				},
				Function: func(rad any) float64 {
					return math.Sin(typeutil.Float(rad))
				},
			}, {
				Name:    `cos`,
				Summary: `Return the cosine of the given number (in radians).`,
				Arguments: []funcArg{
					{
						Name:        `radians`,
						Type:        `float, integer`,
						Description: `The value (in radians) to operate on.`,
					},
				},
				Function: func(rad any) float64 {
					return math.Cos(typeutil.Float(rad))
				},
			}, {
				Name:    `tan`,
				Summary: `Return the tangent of the given number (in radians).`,
				Arguments: []funcArg{
					{
						Name:        `radians`,
						Type:        `float, integer`,
						Description: `The value (in radians) to operate on.`,
					},
				},
				Function: func(rad any) float64 {
					return math.Tan(typeutil.Float(rad))
				},
			}, {
				Name:    `asin`,
				Summary: `Return the arcsine of the given number (in radians).`,
				Arguments: []funcArg{
					{
						Name:        `radians`,
						Type:        `float, integer`,
						Description: `The value (in radians) to operate on.`,
					},
				},
				Function: func(rad any) float64 {
					return math.Asin(typeutil.Float(rad))
				},
			}, {
				Name:    `acos`,
				Summary: `Return the arccosine of the given number (in radians).`,
				Arguments: []funcArg{
					{
						Name:        `radians`,
						Type:        `float, integer`,
						Description: `The value (in radians) to operate on.`,
					},
				},
				Function: func(rad any) float64 {
					return math.Acos(typeutil.Float(rad))
				},
			}, {
				Name:    `atan`,
				Summary: `Return the arctangent of the given number (in radians).`,
				Arguments: []funcArg{
					{
						Name:        `radians`,
						Type:        `float, integer`,
						Description: `The value (in radians) to operate on.`,
					},
				},
				Function: func(rad any) float64 {
					return math.Atan(typeutil.Float(rad))
				},
			}, {
				Name:    `deg2rad`,
				Summary: `Return the given number of degrees in radians.`,
				Arguments: []funcArg{
					{
						Name:        `degrees`,
						Type:        `float, integer`,
						Description: `The value (in degrees) to convert.`,
					},
				},
				Function: func(deg any) float64 {
					return typeutil.Float(deg) * (math.Pi / 180)
				},
			}, {
				Name:    `rad2deg`,
				Summary: `Return the given number of radians in degrees.`,
				Arguments: []funcArg{
					{
						Name:        `radians`,
						Type:        `float, integer`,
						Description: `The value (in radians) to convert.`,
					},
				},
				Function: func(rad any) float64 {
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
		var docName = obj.Name

		switch docName {
		case `minimum_nz`:
			docName = `minimum (excluding zero)`
		case `stddev`:
			docName = `standard deviation`
		}

		group.Functions = append(group.Functions, funcDef{
			Name:    obj.Name,
			Summary: fmt.Sprintf("Return the %s of the given array of numbers.", docName),
			Arguments: []funcArg{
				{
					Name:        `numbers`,
					Type:        `array[float, integer]`,
					Description: `An array of numbers to aggregate.`,
				},
			},
			Function: func(statsFn statsUnaryFn) statsTplFunc {
				return func(in any) (float64, error) {
					var input []float64

					if err := sliceutil.Each(in, func(i int, value any) error {
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
