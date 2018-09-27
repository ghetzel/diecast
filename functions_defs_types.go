package diecast

import (
	"reflect"

	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/timeutil"
	"github.com/ghetzel/go-stockutil/typeutil"
)

func loadStandardFunctionsTypes(rv FuncMap) {
	// fn isBool: Return whether the given *value* is a boolean type.
	rv[`isBool`] = stringutil.IsBoolean

	// fn isInt: Return whether the given *value* is an integer type.
	rv[`isInt`] = stringutil.IsInteger

	// fn isFloat: Return whether the given *value* is a floating-point type.
	rv[`isFloat`] = stringutil.IsFloat

	// fn isZero: Return whether the given *value* is an zero-valued variable.
	rv[`isZero`] = typeutil.IsZero

	// fn isEmpty: Return whether the given *value* is empty.
	rv[`isEmpty`] = typeutil.IsEmpty

	// fn isArray: Return whether the given *value* is an iterable array or slice.
	rv[`isArray`] = typeutil.IsArray

	// fn isMap: Return whether the given *value* is a key-value map type.
	rv[`isMap`] = func(value interface{}) bool {
		return typeutil.IsKind(value, reflect.Map)
	}

	// fn isTime: Return whether the given *value* is parsable as a date/time value.
	rv[`isTime`] = func(value interface{}) bool {
		return !typeutil.V(value).Time().IsZero()
	}

	// fn isDuration: Return whether the given *value* is parsable as a duration.
	rv[`isDuration`] = func(value interface{}) bool {
		return (typeutil.V(value).Duration() != 0)
	}

	// fn autotype: Attempt to automatically determine the type if *value* and return the converted output.
	rv[`autotype`] = stringutil.Autotype

	// fn asStr: Return the *value* as a string.
	rv[`asStr`] = stringutil.ToString

	// fn asInt: Attempt to convert the given *value* to an integer.
	rv[`asInt`] = func(value interface{}) (int64, error) {
		if v, err := stringutil.ConvertToFloat(value); err == nil {
			return int64(v), nil
		} else {
			return 0, err
		}
	}

	// fn asFloat: Attempt to convert the given *value* to a floating-point number.
	rv[`asFloat`] = stringutil.ConvertToFloat

	// fn asBool: Attempt to convert the given *value* to a boolean value.
	rv[`asBool`] = stringutil.ConvertToBool

	// fn asTime: Attempt to parse the given *value* as a date/time value.
	rv[`asTime`] = stringutil.ConvertToTime

	// fn asDuration: Attempt to parse the given *value* as a time duration.
	rv[`asDuration`] = timeutil.ParseDuration

	rv[`s`] = rv[`asStr`]
	rv[`i`] = rv[`asInt`]
	rv[`f`] = rv[`asFloat`]
	rv[`b`] = rv[`asBool`]
	rv[`t`] = rv[`asTime`]
	rv[`d`] = rv[`asDuration`]
}
