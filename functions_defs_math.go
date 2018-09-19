package diecast

import (
	"github.com/ghetzel/go-stockutil/mathutil"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/typeutil"
	"github.com/montanaflynn/stats"
)

func loadStandardFunctionsMath(rv FuncMap) {
	rv[`calc`] = calcFn

	// fn add: Return the sum of all of the given *values*.
	rv[`add`] = func(values ...interface{}) float64 {
		out, _ := calcFn(`+`, values...)
		return out
	}

	// fn subtract: Sequentially subtract all of the given *values*.
	rv[`subtract`] = func(values ...interface{}) float64 {
		out, _ := calcFn(`-`, values...)
		return out
	}

	// fn multiply: Return the product of all of the given *values*.
	rv[`multiply`] = func(values ...interface{}) float64 {
		out, _ := calcFn(`*`, values...)
		return out
	}

	// fn divide: Sequentially divide all of the given *values*.
	rv[`divide`] = func(values ...interface{}) (float64, error) {
		return calcFn(`/`, values...)
	}

	// fn mod: Return the modulus of all of the given *values*.
	rv[`mod`] = func(values ...interface{}) (float64, error) {
		return calcFn(`%`, values...)
	}

	// fn pow: Sequentially exponentiate of all of the given *values*.
	rv[`pow`] = func(values ...interface{}) (float64, error) {
		return calcFn(`^`, values...)
	}

	// fn sequence: Return an array of integers representing a sequence from [0, *n*).
	rv[`sequence`] = func(max interface{}) []int {
		if v, err := stringutil.ConvertToInteger(max); err == nil {
			seq := make([]int, v)

			for i, _ := range seq {
				seq[i] = i
			}

			return seq
		} else {
			return nil
		}
	}

	// fn round: Round a number to the nearest n places.
	rv[`round`] = func(in interface{}, places ...int) (float64, error) {
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
	}

	// fn negate: Return the given number multiplied by -1
	rv[`negate`] = func(value interface{}) float64 {
		return -1 * typeutil.V(value).Float()
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
		rv[fnName] = func(statsFn statsUnary) statsTplFunc {
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
		}(fn)
	}
}
