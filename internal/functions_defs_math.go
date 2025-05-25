package internal

import (
	"fmt"
	"math"

	"github.com/ghetzel/go-stockutil/mathutil"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/typeutil"
	"github.com/montanaflynn/stats"
)

func loadStandardFunctionsMath(funcs FuncMap, server ServerProxy) FuncGroup {
	var group = FuncGroup{
		Name:        `Math and Statistics`,
		Description: `These functions implement basic mathematical and statistical operations on numbers.`,
		Functions: []FuncDef{
			{
				Name:    `calc`,
				Summary: `Perform arbitrary calculations on zero or more numbers.`,
				Arguments: []FuncArg{
					{
						Name:        `operator`,
						Type:        `string`,
						Description: `An operation to perform on the given sequence of numbers.`,
						Valid: []FuncArg{

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
				Examples: []FuncExample{
					{Code: `calc "+" -3 -2 5`, Return: 0},
					{Code: `calc "-" 10 6 4 2`, Return: -2},
					{Code: `calc "*" 5 5 5 5 5`, Return: 3125},
					{Code: `calc "/" 10000 1000 100 10 1`, Return: 0.01},
					{Code: `calc "^" 2 3 4 5 6`, Return: 2.3485425827738332e+108},
					{Code: `calc "%" 54321 12345 4940`, Return: 1},
				},
			}, {
				Name:    `add`,
				Summary: `Return the sum of all of the given values.`,
				Arguments: []FuncArg{
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
				Arguments: []FuncArg{
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
				Arguments: []FuncArg{
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
				Arguments: []FuncArg{
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
				Arguments: []FuncArg{
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
				Arguments: []FuncArg{
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
				Arguments: []FuncArg{
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
				Examples: []FuncExample{
					{Code: `sequence 5`, Return: []int{0, 1, 2, 3, 4}},
					{Code: `sequence 5 1`, Return: []int{1, 2, 3, 4, 5}},
					{Code: `sequence 4 0 3`, Return: []int{0, 3, 6, 9}},
					{Code: `sequence 4 1 3`, Return: []int{1, 4, 7, 10}},
					{Code: `sequence 5 0 0`, Return: []int{0, 0, 0, 0, 0}},
				},
				Function: func(max any, starts ...any) []int {
					var start int = 0
					var step int = 1

					if len(starts) > 0 {
						if !typeutil.IsEmpty(starts[0]) {
							start = int(typeutil.Int(starts[0]))
						}

						if len(starts) > 1 {
							step = int(typeutil.Int(starts[1]))
						}
					}

					if v, err := stringutil.ConvertToInteger(max); err == nil {
						var seq = make([]int, v)

						for i, _ := range seq {
							seq[i] = start + (i * step)
						}

						return seq
					} else {
						return nil
					}
				},
			}, {
				Name:    `round`,
				Summary: `Round a number to the nearest _n_ places.`,
				Arguments: []FuncArg{
					{
						Name:        `number`,
						Type:        `float, integer`,
						Description: `The number to round.`,
					},
				},
				Examples: []FuncExample{
					{Code: `round 0`, Return: 0},
					{Code: `round 1`, Return: 1},
					{Code: `round 1.5537 4`, Return: 1.5537},
					{Code: `round 1.5537 3`, Return: 1.554},
					{Code: `round 1.5537 2`, Return: 1.55},
					{Code: `round 1.5537 1`, Return: 1.6},
					{Code: `round 1.5537 0`, Return: 2},
				},
				Function: func(in any, places ...int) float64 {
					var value = typeutil.Float(in)
					var n = 0

					if len(places) > 0 {
						n = places[0]
					}

					if n > 0 {
						return mathutil.RoundPlaces(value, n)
					} else {
						return mathutil.Round(value)
					}
				},
			}, {
				Name:    `negate`,
				Summary: `Return the given number multiplied by -1.`,
				Arguments: []FuncArg{
					{
						Name:        `number`,
						Type:        `float, integer`,
						Description: `The number to operate on.`,
					},
				},
				Examples: []FuncExample{
					{Code: `negate -1`, Return: 1},
					{Code: `negate 0`, Return: 0},
					{Code: `negate 1`, Return: -1},
				},
				Function: func(value any) float64 {
					if typeutil.IsZero(value) {
						return 0
					}

					return -1 * typeutil.Float(value)
				},
			}, {
				Name:    `isEven`,
				Summary: `Return whether the given number is even.`,
				Arguments: []FuncArg{
					{
						Name:        `number`,
						Type:        `float, integer`,
						Description: `The number to test.`,
					},
				},
				Examples: []FuncExample{
					{Code: `isEven -3`, Return: false},
					{Code: `isEven -2`, Return: true},
					{Code: `isEven -1`, Return: false},
					{Code: `isEven 0`, Return: true},
					{Code: `isEven 1`, Return: false},
					{Code: `isEven 2`, Return: true},
					{Code: `isEven 3`, Return: false},
				},
				Function: func(number any) bool {
					return (math.Mod(typeutil.Float(number), 2) == 0)
				},
			}, {
				Name:    `isOdd`,
				Summary: `Return whether the given number is odd.`,
				Arguments: []FuncArg{
					{
						Name:        `number`,
						Type:        `float, integer`,
						Description: `The number to test.`,
					},
				},
				Examples: []FuncExample{
					{Code: `isOdd -3`, Return: true},
					{Code: `isOdd -2`, Return: false},
					{Code: `isOdd -1`, Return: true},
					{Code: `isOdd 0`, Return: false},
					{Code: `isOdd 1`, Return: true},
					{Code: `isOdd 2`, Return: false},
					{Code: `isOdd 3`, Return: true},
				},
				Function: func(number any) bool {
					return (math.Mod(typeutil.Float(number), 2) != 0)
				},
			}, {
				Name:    `abs`,
				Summary: `Return the absolute value of the given number.`,
				Arguments: []FuncArg{
					{
						Name:        `number`,
						Type:        `float, integer`,
						Description: `The number to operate on.`,
					},
				},
				Examples: []FuncExample{
					{Code: `abs -3`, Return: 3},
					{Code: `abs -2`, Return: 2},
					{Code: `abs -1`, Return: 1},
					{Code: `abs 0`, Return: 0},
					{Code: `abs 1`, Return: 1},
					{Code: `abs 2`, Return: 2},
					{Code: `abs 3`, Return: 3},
				},
				Function: func(number any) float64 {
					return math.Abs(typeutil.Float(number))
				},
			}, {
				Name:    `ceil`,
				Summary: `Return the greatest integer value greater than or equal to the given number.`,
				Arguments: []FuncArg{
					{
						Name:        `number`,
						Type:        `float, integer`,
						Description: `The number to operate on.`,
					},
				},
				Examples: []FuncExample{
					{Code: `ceil -3.1`, Return: -3},
					{Code: `ceil -2.1`, Return: -2},
					{Code: `ceil -1.1`, Return: -1},
					{Code: `ceil 0`, Return: 0},
					{Code: `ceil 1.1`, Return: 2},
					{Code: `ceil 2.1`, Return: 3},
					{Code: `ceil 3.1`, Return: 4},
				},
				Function: func(number any) float64 {
					return math.Ceil(typeutil.Float(number))
				},
			}, {
				Name:    `floor`,
				Summary: `Return the greatest integer value less than or equal to the given number.`,
				Arguments: []FuncArg{
					{
						Name:        `number`,
						Type:        `float, integer`,
						Description: `The number to operate on.`,
					},
				},
				Examples: []FuncExample{
					{Code: `floor -3.1`, Return: -4},
					{Code: `floor -2.1`, Return: -3},
					{Code: `floor -1.1`, Return: -2},
					{Code: `floor 0`, Return: 0},
					{Code: `floor 1.1`, Return: 1},
					{Code: `floor 2.1`, Return: 2},
					{Code: `floor 3.1`, Return: 3},
				},
				Function: func(number any) float64 {
					return math.Floor(typeutil.Float(number))
				},
			}, {
				Name:    `sin`,
				Summary: `Return the sine of the given number (in radians).`,
				Arguments: []FuncArg{
					{
						Name:        `radians`,
						Type:        `float, integer`,
						Description: `The value (in radians) to operate on.`,
					},
				},
				Examples: []FuncExample{
					{Code: `sin 0`, Return: 0},
					{Code: `sin 0.5`, Return: 0.479425538604203},
					{Code: `sin 1.5707963268`, Return: 1},
				},
				Function: func(rad any) float64 {
					return math.Sin(typeutil.Float(rad))
				},
			}, {
				Name:    `cos`,
				Summary: `Return the cosine of the given number (in radians).`,
				Arguments: []FuncArg{
					{
						Name:        `radians`,
						Type:        `float, integer`,
						Description: `The value (in radians) to operate on.`,
					},
				},
				Examples: []FuncExample{
					{Code: `cos 0`, Return: 1},
					{Code: `cos 0.5`, Return: 0.8775825618903728},
					{Code: `cos 1`, Return: 0.5403023058681398},
				},
				Function: func(rad any) float64 {
					return math.Cos(typeutil.Float(rad))
				},
			}, {
				Name:    `tan`,
				Summary: `Return the tangent of the given number (in radians).`,
				Arguments: []FuncArg{
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
				Arguments: []FuncArg{
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
				Arguments: []FuncArg{
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
				Arguments: []FuncArg{
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
				Arguments: []FuncArg{
					{
						Name:        `degrees`,
						Type:        `float, integer`,
						Description: `The value (in degrees) to convert.`,
					},
				},
				Examples: []FuncExample{
					{Code: `deg2rad 0`, Return: 0},
					{Code: `deg2rad 90`, Return: math.Pi / 2},
					{Code: `deg2rad 180`, Return: math.Pi},
					{Code: `deg2rad 270`, Return: (3 * math.Pi) / 2},
					{Code: `deg2rad 360`, Return: 2 * math.Pi},
				},
				Function: func(deg any) float64 {
					return typeutil.Float(deg) * (math.Pi / 180)
				},
			}, {
				Name:    `rad2deg`,
				Summary: `Return the given number of radians in degrees.`,
				Arguments: []FuncArg{
					{
						Name:        `radians`,
						Type:        `float, integer`,
						Description: `The value (in radians) to convert.`,
					},
				},
				Examples: []FuncExample{
					{Code: `rad2deg 6.2831853072`, Return: 360},
					{Code: `rad2deg 3.141592653`, Return: 180},
					{Code: `rad2deg 1.5707963268`, Return: 90},
					{Code: `rad2deg 0`, Return: 0},
					{Code: `rad2deg 6.2831853072 10`, Return: 360.0000000012},
					{Code: `rad2deg 3.141592653 10`, Return: 179.9999999662},
					{Code: `rad2deg 1.5707963268 10`, Return: 90.0000000003},
					{Code: `rad2deg 0 10`, Return: 0},
				},
				Function: func(rad any, places ...any) float64 {
					var rplaces int = 0

					if len(places) > 0 {
						rplaces = typeutil.NInt(places[0])
					}

					return mathutil.RoundPlaces(
						typeutil.Float(rad)*(180/math.Pi),
						rplaces,
					)
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

		group.Functions = append(group.Functions, FuncDef{
			Name:    obj.Name,
			Summary: fmt.Sprintf("Return the %s of the given array of numbers.", docName),
			Arguments: []FuncArg{
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
