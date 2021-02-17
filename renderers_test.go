package diecast

import (
	"bytes"
	"io/ioutil"
	"net/http/httptest"
	"testing"

	"github.com/alecthomas/assert"
)

func TestRendererConfigWithResponse(t *testing.T) {
	var a = RendererConfig{}

	assert.Nil(t, a.Request)
	assert.Nil(t, a.Data)

	var b = a.WithResponse(
		ioutil.NopCloser(bytes.NewBufferString(`hello`)),
		httptest.NewRequest(`GET`, `/`, nil),
	)

	assert.Nil(t, a.Request)
	assert.Nil(t, a.Data)

	assert.NotNil(t, b.Request)
	assert.NotNil(t, b.Data)
}
