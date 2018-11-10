package diecast

import (
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/stringutil"
)

func loadStandardFunctionsMisc(rv FuncMap) {
	// fn eqx: A relaxed-type version of the **eq** builtin function.
	rv[`eqx`] = stringutil.RelaxedEqual

	// fn nex: A relaxed-type version of the **ne** builtin function.
	rv[`nex`] = func(first interface{}, second interface{}) (bool, error) {
		eq, err := stringutil.RelaxedEqual(first, second)
		return !eq, err
	}

	// fn switch: Provide a simple inline switch-case style decision mechanism.
	rv[`switch`] = func(input interface{}, fallback interface{}, pairs ...interface{}) interface{} {
		for _, pair := range sliceutil.Chunks(pairs, 2) {
			if len(pair) == 2 {
				if eq, err := stringutil.RelaxedEqual(input, pair[0]); err == nil && eq {
					return pair[1]
				}
			}
		}

		return fallback
	}
}
