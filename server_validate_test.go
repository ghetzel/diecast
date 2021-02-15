package diecast

import (
	"net/http/httptest"
	"testing"

	"github.com/alecthomas/assert"
)

func TestValidatorConfigShouldValidateRequest(t *testing.T) {
	var vc = new(ValidatorConfig)
	var req = httptest.NewRequest(`GET`, `/hello/there.html`, nil)

	assert.True(t, vc.ShouldValidateRequest(req))

	vc.Only = `/hello/there.html`
	assert.True(t, vc.ShouldValidateRequest(req))

	req = httptest.NewRequest(`GET`, `/other.html`, nil)
	assert.False(t, vc.ShouldValidateRequest(req))

	vc.Only = nil
	vc.Except = []string{
		`*.yaml`,
		`*.json`,
	}

	req = httptest.NewRequest(`GET`, `/file.html`, nil)
	assert.True(t, vc.ShouldValidateRequest(req))

	req = httptest.NewRequest(`GET`, `/file.yaml`, nil)
	assert.False(t, vc.ShouldValidateRequest(req))

	req = httptest.NewRequest(`GET`, `/file.json`, nil)
	assert.False(t, vc.ShouldValidateRequest(req))

}
