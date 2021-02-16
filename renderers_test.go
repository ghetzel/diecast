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
	assert.Nil(t, a.Response)
	assert.Nil(t, a.Data)

	var b = a.WithResponse(
		httptest.NewRecorder(),
		httptest.NewRequest(`GET`, `/`, nil),
		ioutil.NopCloser(bytes.NewBufferString(`hello`)),
	)

	assert.Nil(t, a.Request)
	assert.Nil(t, a.Response)
	assert.Nil(t, a.Data)

	assert.NotNil(t, b.Request)
	assert.NotNil(t, b.Response)
	assert.NotNil(t, b.Data)
}
