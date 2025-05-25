package diecast

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/ghetzel/diecast/v2/internal"
	"github.com/ghetzel/testify/require"
)

func TestFunctionExamples(t *testing.T) {
	var assert = require.New(t)

	var fgroups, _ = internal.GetFunctions(nil)

	for _, group := range fgroups {
		if group.SkipTest {
			continue
		}

		for _, def := range group.Functions {
			if def.SkipTest {
				continue
			}

			for _, example := range def.Examples {
				if example.SkipTest {
					continue
				}

				var msg = example.Code
				var ctx = NewContext(nil)
				ctx.SetValue(`input`, example.Input)

				var tmpl, err = ParseTemplateWithFuncs(
					bytes.NewBufferString(`{{ `+example.Code+` }}`),
					internal.GetStandardFunctions(nil),
				)

				assert.NoError(err, msg)
				assert.NotNil(tmpl, msg)

				tmpl.SetContext(ctx)
				assert.EqualValues(fmt.Sprintf("%v", example.Return), tmpl.String(), msg)
			}
		}
	}
}
