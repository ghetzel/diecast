package diecast

import (
	"net/http/httptest"
	"testing"

	"github.com/alecthomas/assert"
)

func TestWithRequest(t *testing.T) {
	var a = ValidatorConfig{}

	assert.Nil(t, a.Request)

	var b = a.WithRequest(httptest.NewRequest(`GET`, `/`, nil))

	assert.Nil(t, a.Request)
	assert.NotNil(t, b.Request)
}
