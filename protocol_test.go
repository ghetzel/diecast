package diecast

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProtocolRequest(t *testing.T) {
	assert := require.New(t)
	fns := GetStandardFunctions(nil)

	req := &ProtocolRequest{
		TemplateData: map[string]interface{}{
			`a`: 123,
		},
		TemplateFuncs: fns,
	}

	assert.EqualValues(123, req.Template(`{{ 123 }}`).Auto())
	assert.EqualValues([]string{
		`1`, `2`, `3`,
	}, req.Template([]string{
		`1`, `2`, `3`,
	}).Value)
}
