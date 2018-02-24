package diecast

import "github.com/ghetzel/go-stockutil/stringutil"

func loadStandardFunctionsMisc(rv FuncMap) {
	// fn eqx: A relaxed-type version of the **eq** builtin function.
	rv[`eqx`] = stringutil.RelaxedEqual

	// fn nex: A relaxed-type version of the **ne** builtin function.
	rv[`nex`] = func(first interface{}, second interface{}) (bool, error) {
		eq, err := stringutil.RelaxedEqual(first, second)
		return !eq, err
	}
}
