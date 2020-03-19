package diecast

import (
	"testing"

	"github.com/ghetzel/testify/require"
)

func TestProtocolRequest(t *testing.T) {
	var assert = require.New(t)
	var fns = GetStandardFunctions(nil)

	var req = &ProtocolRequest{
		TemplateData: map[string]interface{}{
			`a`: 123,
		},
		TemplateFuncs: fns,
	}

	v, err := req.Template(`{{ 123 }}`)
	assert.NoError(err)
	assert.EqualValues(123, v.Auto())

	v, err = req.Template([]string{
		`1`, `2`, `3`,
	})

	assert.NoError(err)
	assert.EqualValues([]string{
		`1`, `2`, `3`,
	}, v.Value)
}
