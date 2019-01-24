package diecast

import (
	"fmt"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/typeutil"
	"regexp"
)

func loadStandardFunctionsComparisons() funcGroup {
	return funcGroup{
		Name:        `Comparison and Conditionals`,
		Description: `Used for comparing two values, or for returning one of many options based on a given condition.`,
		Functions: []funcDef{
			{}, {
				Name:     `eqx`,
				Summary:  `Return whether two values are equal (any type).`,
				Function: stringutil.RelaxedEqual,
			}, {
				Name:    `nex`,
				Summary: `Return whether two values are not equal (any type).`,
				Function: func(first interface{}, second interface{}) (bool, error) {
					eq, err := stringutil.RelaxedEqual(first, second)
					return !eq, err
				},
			}, {
				Name:    `gtx`,
				Summary: `Return whether the first value is numerically or lexically greater than the second.`,
				Function: func(first interface{}, second interface{}) (bool, error) {
					return cmp(`ge`, first, second)
				},
			}, {
				Name:    `gex`,
				Summary: `Return whether the first value is numerically or lexically greater than or equal to the second.`,
				Function: func(first interface{}, second interface{}) (bool, error) {
					return cmp(`ge`, first, second)
				},
			}, {
				Name:    `ltx`,
				Summary: `Return whether the first value is numerically or lexically less than the second.`,
				Function: func(first interface{}, second interface{}) (bool, error) {
					return cmp(`lt`, first, second)
				},
			}, {
				Name:    `lex`,
				Summary: `Return whether the first value is numerically or lexically less than or equal to the second.`,
				Function: func(first interface{}, second interface{}) (bool, error) {
					return cmp(`le`, first, second)
				},
			}, {
				Name:    `compare`,
				Summary: `A generic comparison function. Accepts operators: "gt", "ge", "lt", "le", "eq", "ne"`,
				Function: func(operator string, first interface{}, second interface{}) (bool, error) {
					switch operator {
					case `gt`, `ge`, `lt`, `le`:
						return cmp(operator, first, second)
					case `eq`:
						return stringutil.RelaxedEqual(first, second)
					case `ne`:
						eq, err := stringutil.RelaxedEqual(first, second)
						return !eq, err
					default:
						return false, fmt.Errorf("Invalid operator %q", operator)
					}
				},
			}, {
				Name:    `match`,
				Summary: `Return whether the given value matches the given regular expression.`,
				Function: func(pattern string, value interface{}) (bool, error) {
					if rx, err := regexp.Compile(pattern); err == nil {
						return rx.MatchString(typeutil.String(value)), nil
					} else {
						return false, err
					}
				},
			}, {
				Name:    `switch`,
				Summary: `Provide a simple inline switch-case style decision mechanism.`,
				Function: func(input interface{}, fallback interface{}, pairs ...interface{}) interface{} {
					for _, pair := range sliceutil.Chunks(pairs, 2) {
						if len(pair) == 2 {
							if eq, err := stringutil.RelaxedEqual(input, pair[0]); err == nil && eq {
								return pair[1]
							}
						}
					}

					return fallback
				},
			},
		},
	}
}
