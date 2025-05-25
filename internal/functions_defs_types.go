package internal

import (
	"reflect"
	"time"

	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/timeutil"
	"github.com/ghetzel/go-stockutil/typeutil"
)

func loadStandardFunctionsTypes(funcs FuncMap, server ServerProxy) FuncGroup {
	var group = FuncGroup{
		Name:        `Type Detection and Manipulation`,
		Description: `Used to detect and convert discrete values into different data types.`,
		Functions: []FuncDef{
			{
				Name:     `isBool`,
				Summary:  `Return whether the given *value* is a boolean type.`,
				Function: stringutil.IsBoolean,
				Examples: []FuncExample{
					{Code: `isBool true`, Return: true},
					{Code: `isBool "true"`, Return: true},
					{Code: `isBool "True"`, Return: true},
					{Code: `isBool 1`, Return: false},
					{Code: `isBool "hello"`, Return: false},
					{Code: `isBool false`, Return: true},
					{Code: `isBool "false"`, Return: true},
					{Code: `isBool 0`, Return: false},
					{Code: `isBool ""`, Return: false},
				},
			}, {
				Name:     `isInt`,
				Summary:  `Return whether the given *value* is an integer type.`,
				Function: stringutil.IsInteger,
				Examples: []FuncExample{
					{Code: `isInt 0`, Return: true},
					{Code: `isInt 1`, Return: true},
					{Code: `isInt "0"`, Return: true},
					{Code: `isInt "1"`, Return: true},
					{Code: `isInt "-1"`, Return: true},
					{Code: `isInt false`, Return: false},
					{Code: `isInt true`, Return: false},
					{Code: `isInt "seven"`, Return: false},
					{Code: `isInt 3.14`, Return: false},
				},
			}, {
				Name:     `isFloat`,
				Summary:  `Return whether the given *value* is a floating-point type.`,
				Function: stringutil.IsFloat,
				Examples: []FuncExample{
					{Code: `isFloat 0`, Return: true},
					{Code: `isFloat 1`, Return: true},
					{Code: `isFloat "0"`, Return: true},
					{Code: `isFloat "1"`, Return: true},
					{Code: `isFloat "-1.1"`, Return: true},
					{Code: `isFloat 3.14`, Return: true},
					{Code: `isFloat 3.00`, Return: true},
					{Code: `isFloat "seven"`, Return: false},
				},
			}, {
				Name:     `isZero`,
				Summary:  `Return whether the given *value* is an zero-valued variable.`,
				Function: typeutil.IsZero,
				Examples: []FuncExample{
					{Code: `isZero 0`, Return: true},
					{Code: `isZero 1`, Return: false},
					{Code: `isZero ""`, Return: true},
					{Code: `isZero "0"`, Return: false},
					{Code: `isZero true`, Return: false},
					{Code: `isZero false`, Return: true},
					{Code: `isZero "true"`, Return: false},
					{Code: `isZero "false"`, Return: false},
					{Code: `isZero 0.0`, Return: true},
				},
			}, {
				Name:     `isEmpty`,
				Summary:  `Return whether the given *value* is empty.`,
				Function: typeutil.IsEmpty,
				Examples: []FuncExample{
					{Code: `isEmpty ""`, Return: true},
					{Code: `isEmpty $.input`, Return: false, Input: nil},
					{Code: `isEmpty $.input`, Return: true, Input: make([]string, 0)},
					{Code: `isEmpty 0`, Return: false},
					{Code: `isEmpty false`, Return: false},
				},
			}, {
				Name:    `isNotZero`,
				Summary: `Return whether the given *value* is NOT a zero-valued variable.`,
				Function: func(value any) bool {
					return !typeutil.IsZero(value)
				},
				Examples: []FuncExample{
					{Code: `isNotZero 0`, Return: false},
					{Code: `isNotZero 1`, Return: true},
					{Code: `isNotZero ""`, Return: false},
					{Code: `isNotZero "0"`, Return: true},
					{Code: `isNotZero true`, Return: true},
					{Code: `isNotZero false`, Return: false},
					{Code: `isNotZero "true"`, Return: true},
					{Code: `isNotZero "false"`, Return: true},
					{Code: `isNotZero 0.0`, Return: false},
				},
			}, {
				Name:    `isNotEmpty`,
				Summary: `Return whether the given *value* is NOT empty.`,
				Function: func(value any) bool {
					return !typeutil.IsEmpty(value)
				},
				Examples: []FuncExample{
					{Code: `isNotEmpty ""`, Return: false},
					{Code: `isNotEmpty $.input`, Return: true, Input: nil},
					{Code: `isNotEmpty $.input`, Return: false, Input: make([]string, 0)},
					{Code: `isNotEmpty 0`, Return: true},
					{Code: `isNotEmpty false`, Return: true},
				},
			}, {
				Name:     `isArray`,
				Summary:  `Return whether the given *value* is an iterable array or slice.`,
				Function: typeutil.IsArray,
				Examples: []FuncExample{
					{Code: `isArray ""`, Return: false},
					{Code: `isArray 0`, Return: false},
					{Code: `isArray $.input`, Return: false, Input: nil},
					{Code: `isArray $.input`, Return: true, Input: make([]int, 0)},
					{Code: `isArray $.input`, Return: true, Input: []int{1, 2, 3}},
					{Code: `isArray $.input`, Return: false, Input: make(map[string]any)},
				},
			}, {
				Name:    `isMap`,
				Summary: `Return whether the given *value* is a key-value map type.`,
				Function: func(value any) bool {
					return typeutil.IsKind(value, reflect.Map)
				},
				Examples: []FuncExample{
					{Code: `isMap ""`, Return: false},
					{Code: `isMap 0`, Return: false},
					{Code: `isMap $.input`, Return: false, Input: nil},
					{Code: `isMap $.input`, Return: false, Input: make([]int, 0)},
					{Code: `isMap $.input`, Return: false, Input: []int{1, 2, 3}},
					{Code: `isMap $.input`, Return: false, Input: make(map[string]any)},
					{Code: `isMap $.input`, Return: true, Input: map[string]any{`a`: 1, `b`: 2}},
				},
			}, {
				Name:    `isTime`,
				Summary: `Return whether the given *value* is parsable as a date/time value.`,
				Function: func(value any) bool {
					return !typeutil.V(value).Time().IsZero()
				},
				Examples: []FuncExample{
					{Code: `isTime ""`, Return: false},
					{Code: `isTime 0`, Return: true},
					{Code: `isTime $.input`, Return: false, Input: nil},
					{Code: `isTime $.input`, Return: true, Input: time.Now().UTC()},
					{Code: `isTime "2006-01-02T15:14:03-07:00"`, Return: true},
					{Code: `isTime "123s"`, Return: false},
				},
			}, {
				Name:    `isDuration`,
				Summary: `Return whether the given *value* is parsable as a duration.`,
				Function: func(value any) bool {
					return (typeutil.V(value).Duration() != 0)
				},
				Examples: []FuncExample{
					{Code: `isDuration ""`, Return: false},
					{Code: `isDuration 0`, Return: false},
					{Code: `isDuration $.input`, Return: false, Input: nil},
					{Code: `isDuration "2006-01-02T15:04:03-07:00"`, Return: false},
					{Code: `isDuration "123s"`, Return: true},
				},
			}, {
				Name:     `autotype`,
				Summary:  `Attempt to automatically determine the type if *value* and return the converted output.`,
				Function: stringutil.Autotype,
				Examples: []FuncExample{
					{Code: `autotype "hi"`, Return: `hi`},
					{Code: `autotype "0"`, Return: int64(0)},
					{Code: `autotype "true"`, Return: true},
					{Code: `autotype false`, Return: false},
				},
			}, {
				Name:     `asStr`,
				Aliases:  []string{`s`},
				Summary:  `Return the *value* as a string.`,
				Function: stringutil.ToString,
				Examples: []FuncExample{
					{Code: `asStr "hi"`, Return: `hi`},
					{Code: `asStr "0"`, Return: `0`},
					{Code: `asStr true`, Return: `true`},
					{Code: `asStr false`, Return: `false`},
					{Code: `asStr $.input`, Return: `2m3s`, Input: 123 * time.Second},
				},
			}, {
				Name:    `asInt`,
				Aliases: []string{`i`},
				Summary: `Attempt to convert the given *value* to an integer.`,
				Function: func(value any) int64 {
					return typeutil.Int(value)
				},
				Examples: []FuncExample{
					{Code: `asInt "hi"`, Return: 0},
					{Code: `asInt "0"`, Return: 0},
					{Code: `asInt true`, Return: 0},
					{Code: `asInt false`, Return: 0},
					{Code: `asInt 3.14`, Return: 3},
				},
			}, {
				Name:    `asFloat`,
				Aliases: []string{`f`},
				Summary: `Attempt to convert the given *value* to a floating-point number.`,
				Function: func(value any) float64 {
					return typeutil.Float(value)
				},
				Examples: []FuncExample{
					{Code: `asFloat "hi"`, Return: 0.0},
					{Code: `asFloat "0"`, Return: 0.0},
					{Code: `asFloat true`, Return: 0.0},
					{Code: `asFloat false`, Return: 0.0},
					{Code: `asFloat 3.14`, Return: 3.14},
					{Code: `asFloat 3`, Return: 3.0},
				},
			}, {
				Name:    `asBool`,
				Aliases: []string{`b`},
				Summary: `Attempt to convert the given *value* to a boolean value.`,
				Function: func(value any) bool {
					return typeutil.Bool(value)
				},
				Examples: []FuncExample{
					{Code: `asBool "hi"`, Return: true},
					{Code: `asBool "0"`, Return: false},
					{Code: `asBool "1"`, Return: true},
					{Code: `asBool true`, Return: true},
					{Code: `asBool false`, Return: false},
					{Code: `asBool 0`, Return: false},
					{Code: `asBool 1`, Return: true},
					{Code: `asBool 3.14`, Return: true},
					{Code: `asBool 3`, Return: true},
				},
			}, {
				Name:     `asTime`,
				Aliases:  []string{`t`},
				Summary:  `Attempt to parse the given *value* as a date/time value.`,
				Function: stringutil.ConvertToTime,
			}, {
				Name:     `asDuration`,
				Aliases:  []string{`d`},
				Summary:  `Attempt to parse the given *value* as a time duration.`,
				Function: timeutil.ParseDuration,
			},
		},
	}

	group.Functions = append(group.Functions, []FuncDef{
		{
			Name:     `s`,
			Alias:    `asStr`,
			Hidden:   true,
			Function: group.fn(`asStr`),
		}, {
			Name:     `i`,
			Alias:    `asInt`,
			Hidden:   true,
			Function: group.fn(`asInt`),
		}, {
			Name:     `f`,
			Alias:    `asFloat`,
			Hidden:   true,
			Function: group.fn(`asFloat`),
		}, {
			Name:     `b`,
			Alias:    `asBool`,
			Hidden:   true,
			Function: group.fn(`asBool`),
		}, {
			Name:     `t`,
			Alias:    `asTime`,
			Hidden:   true,
			Function: group.fn(`asTime`),
		}, {
			Name:     `d`,
			Alias:    `asDuration`,
			Hidden:   true,
			Function: group.fn(`asDuration`),
		},
	}...)

	return group
}
