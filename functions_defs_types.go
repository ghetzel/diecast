package diecast

import (
	"reflect"

	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/timeutil"
	"github.com/ghetzel/go-stockutil/typeutil"
)

func loadStandardFunctionsTypes(funcs FuncMap) funcGroup {
	group := funcGroup{
		Name:        `Type Detection and Manipulation`,
		Description: `Used to detect and convert discrete values into different data types.`,
		Functions: []funcDef{
			{
				Name:     `isBool`,
				Summary:  `Return whether the given *value* is a boolean type.`,
				Function: stringutil.IsBoolean,
			}, {
				Name:     `isInt`,
				Summary:  `Return whether the given *value* is an integer type.`,
				Function: stringutil.IsInteger,
			}, {
				Name:     `isFloat`,
				Summary:  `Return whether the given *value* is a floating-point type.`,
				Function: stringutil.IsFloat,
			}, {
				Name:     `isZero`,
				Summary:  `Return whether the given *value* is an zero-valued variable.`,
				Function: typeutil.IsZero,
			}, {
				Name:     `isEmpty`,
				Summary:  `Return whether the given *value* is empty.`,
				Function: typeutil.IsEmpty,
			}, {
				Name:     `isArray`,
				Summary:  `Return whether the given *value* is an iterable array or slice.`,
				Function: typeutil.IsArray,
			}, {
				Name:    `isMap`,
				Summary: `Return whether the given *value* is a key-value map type.`,
				Function: func(value interface{}) bool {
					return typeutil.IsKind(value, reflect.Map)
				},
			}, {
				Name:    `isTime`,
				Summary: `Return whether the given *value* is parsable as a date/time value.`,
				Function: func(value interface{}) bool {
					return !typeutil.V(value).Time().IsZero()
				},
			}, {
				Name:    `isDuration`,
				Summary: `Return whether the given *value* is parsable as a duration.`,
				Function: func(value interface{}) bool {
					return (typeutil.V(value).Duration() != 0)
				},
			}, {
				Name:     `autotype`,
				Summary:  `Attempt to automatically determine the type if *value* and return the converted output.`,
				Function: stringutil.Autotype,
			}, {
				Name:     `asStr`,
				Summary:  `Return the *value* as a string.`,
				Function: stringutil.ToString,
			}, {
				Name:    `asInt`,
				Summary: `Attempt to convert the given *value* to an integer.`,
				Function: func(value interface{}) (int64, error) {
					if v, err := stringutil.ConvertToFloat(value); err == nil {
						return int64(v), nil
					} else {
						return 0, err
					}
				},
			}, {
				Name:     `asFloat`,
				Summary:  `Attempt to convert the given *value* to a floating-point number.`,
				Function: stringutil.ConvertToFloat,
			}, {
				Name:     `asBool`,
				Summary:  `Attempt to convert the given *value* to a boolean value.`,
				Function: stringutil.ConvertToBool,
			}, {
				Name:     `asTime`,
				Summary:  `Attempt to parse the given *value* as a date/time value.`,
				Function: stringutil.ConvertToTime,
			}, {
				Name:     `asDuration`,
				Summary:  `Attempt to parse the given *value* as a time duration.`,
				Function: timeutil.ParseDuration,
			},
		},
	}

	group.Functions = append(group.Functions, []funcDef{
		{
			Name:     `s`,
			Alias:    `asStr`,
			Function: group.fn(`asStr`),
		}, {
			Name:     `i`,
			Alias:    `asInt`,
			Function: group.fn(`asInt`),
		}, {
			Name:     `f`,
			Alias:    `asFloat`,
			Function: group.fn(`asFloat`),
		}, {
			Name:     `b`,
			Alias:    `asBool`,
			Function: group.fn(`asBool`),
		}, {
			Name:     `t`,
			Alias:    `asTime`,
			Function: group.fn(`asTime`),
		}, {
			Name:     `d`,
			Alias:    `asDuration`,
			Function: group.fn(`asDuration`),
		},
	}...)

	return group
}
