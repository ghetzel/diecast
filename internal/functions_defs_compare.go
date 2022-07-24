package internal

import (
	"fmt"
	"regexp"

	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/typeutil"
)

func loadStandardFunctionsComparisons(funcs FuncMap, server ServerProxy) FuncGroup {
	return FuncGroup{
		Name:        `Comparison Functions`,
		Description: `Used for comparing two values, or for returning one of many options based on a given condition.`,
		Functions: []FuncDef{
			{
				Name:    `eqx`,
				Summary: `Return whether two values are equal (any type).`,
				Arguments: []FuncArg{
					{
						Name:        `left`,
						Type:        `any`,
						Description: `The first value.`,
					}, {
						Name:        `right`,
						Type:        `any`,
						Description: `The second value.`,
					},
				},
				Examples: []FuncExample{
					{
						Code:   `eqx "1" "1"`,
						Return: true,
					}, {
						Code:   `eqx "1" "2"`,
						Return: false,
					}, {
						Code:   `eqx "1" 1`,
						Return: true,
					}, {
						Code:   `eqx 1 1.0`,
						Return: true,
					}, {
						Code:   `eqx "1" 2`,
						Return: false,
					},
				},
				Function: stringutil.RelaxedEqual,
			}, {
				Name:    `nex`,
				Summary: `Return whether two values are not equal (any type).`,
				Arguments: []FuncArg{
					{
						Name:        `left`,
						Type:        `any`,
						Description: `The first value.`,
					}, {
						Name:        `right`,
						Type:        `any`,
						Description: `The second value.`,
					},
				},
				Examples: []FuncExample{
					{
						Code:   `nex "1" "1"`,
						Return: false,
					}, {
						Code:   `nex "1" "2"`,
						Return: true,
					}, {
						Code:   `nex "1" 1`,
						Return: false,
					}, {
						Code:   `nex 1 1.0`,
						Return: false,
					}, {
						Code:   `nex "1" 2`,
						Return: true,
					},
				},
				Function: func(first interface{}, second interface{}) (bool, error) {
					eq, err := stringutil.RelaxedEqual(first, second)
					return !eq, err
				},
			}, {
				Name:    `gtx`,
				Summary: `Return whether the first value is numerically or lexically greater than the second.`,
				Arguments: []FuncArg{
					{
						Name:        `left`,
						Type:        `any`,
						Description: `The first value.`,
					}, {
						Name:        `right`,
						Type:        `any`,
						Description: `The second value.`,
					},
				},
				Function: func(first interface{}, second interface{}) (bool, error) {
					return cmp(`ge`, first, second)
				},
			}, {
				Name:    `gex`,
				Summary: `Return whether the first value is numerically or lexically greater than or equal to the second.`,
				Arguments: []FuncArg{
					{
						Name:        `left`,
						Type:        `any`,
						Description: `The first value.`,
					}, {
						Name:        `right`,
						Type:        `any`,
						Description: `The second value.`,
					},
				},
				Function: func(first interface{}, second interface{}) (bool, error) {
					return cmp(`ge`, first, second)
				},
			}, {
				Name:    `ltx`,
				Summary: `Return whether the first value is numerically or lexically less than the second.`,
				Arguments: []FuncArg{
					{
						Name:        `left`,
						Type:        `any`,
						Description: `The first value.`,
					}, {
						Name:        `right`,
						Type:        `any`,
						Description: `The second value.`,
					},
				},
				Function: func(first interface{}, second interface{}) (bool, error) {
					return cmp(`lt`, first, second)
				},
			}, {
				Name:    `lex`,
				Summary: `Return whether the first value is numerically or lexically less than or equal to the second.`,
				Arguments: []FuncArg{
					{
						Name:        `left`,
						Type:        `any`,
						Description: `The first value.`,
					}, {
						Name:        `right`,
						Type:        `any`,
						Description: `The second value.`,
					},
				},
				Function: func(first interface{}, second interface{}) (bool, error) {
					return cmp(`le`, first, second)
				},
			}, {
				Name:    `compare`,
				Summary: `A generic comparison function. Accepts operators: "gt", "ge", "lt", "le", "eq", "ne"`,
				Arguments: []FuncArg{
					{
						Name:        `operator`,
						Type:        `string`,
						Description: `The type of compare operation being performed.`,
					}, {
						Name:        `left`,
						Type:        `any`,
						Description: `The first value.`,
					}, {
						Name:        `right`,
						Type:        `any`,
						Description: `The second value.`,
					},
				},
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
				Arguments: []FuncArg{
					{
						Name:        `pattern`,
						Type:        `string, array`,
						Description: `The regular expression to match with, or an array of regular expressions (any of which may match).`,
					}, {
						Name:        `value`,
						Type:        `string`,
						Description: `The value to match against.`,
					},
				},
				Function: func(patterns interface{}, value interface{}) (bool, error) {
					for _, pattern := range sliceutil.Stringify(patterns) {
						if rx, err := regexp.Compile(pattern); err == nil {
							if rx.MatchString(typeutil.String(value)) {
								return true, nil
							}
						} else {
							return false, err
						}
					}

					return false, nil
				},
			}, {
				Name:    `switch`,
				Summary: `Provide a simple inline switch-case style decision mechanism.`,
				Arguments: []FuncArg{
					{
						Name:        `input`,
						Type:        `any`,
						Description: `The value being tested.`,
					}, {
						Name:        `fallback`,
						Type:        `any`,
						Description: `The "default" value if none of the subsequent conditions match.`,
					}, {
						Name: `criteria`,
						Type: `array[if, then]`,
						Description: `An array of values representing possible values of _input_, and the value to ` +
							`return if input matches. Arguments are consumed as an array of value-result pairs.`,
					},
				},
				Examples: []FuncExample{
					{
						Code:   `switch "yellow" "danger" "yellow" "warning" "green" "success" "blue" "info"`,
						Return: `warning`,
					}, {
						Code:   `switch "green" "danger" "yellow" "warning" "green" "success" "blue" "info"`,
						Return: `success`,
					}, {
						Code:   `switch "blue" "danger" "yellow" "warning" "green" "success" "blue" "info"`,
						Return: `info`,
					}, {
						Code:   `switch "red" "danger" "yellow" "warning" "green" "success" "blue" "info"`,
						Return: `danger`,
					}, {
						Code:   `switch "potato" "danger" "yellow" "warning" "green" "success" "blue" "info"`,
						Return: `danger`,
					},
				},
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
