package diecast

import (
	"fmt"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/typeutil"
	"regexp"
)

func loadStandardFunctionsMisc(rv FuncMap) {
	// fn eqx: A relaxed-type version of the **eq** builtin function.
	rv[`eqx`] = stringutil.RelaxedEqual

	// fn nex: A relaxed-type version of the **ne** builtin function.
	rv[`nex`] = func(first interface{}, second interface{}) (bool, error) {
		eq, err := stringutil.RelaxedEqual(first, second)
		return !eq, err
	}

	// fn gtx: A relaxed-type replacement for the **gtx** builtin function.
	rv[`gtx`] = func(first interface{}, second interface{}) (bool, error) {
		return cmp(`ge`, first, second)
	}

	// fn gex: A relaxed-type replacement for the **gex** builtin function.
	rv[`gex`] = func(first interface{}, second interface{}) (bool, error) {
		return cmp(`ge`, first, second)
	}

	// fn ltx: A relaxed-type replacement for the **ltx** builtin function.
	rv[`ltx`] = func(first interface{}, second interface{}) (bool, error) {
		return cmp(`lt`, first, second)
	}

	// fn lex: A relaxed-type replacement for the **lex** builtin function.
	rv[`lex`] = func(first interface{}, second interface{}) (bool, error) {
		return cmp(`le`, first, second)
	}

	// fn compare: Ageneric comparison function. Accepts operators: "gt", "ge", "lt", "le", "eq", "ne"
	rv[`compare`] = func(operator string, first interface{}, second interface{}) (bool, error) {
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
	}

	// fn match: Return whether the given value matches the given regular expression.
	rv[`match`] = func(pattern string, value interface{}) (bool, error) {
		if rx, err := regexp.Compile(pattern); err == nil {
			return rx.MatchString(typeutil.String(value)), nil
		} else {
			return false, err
		}
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
