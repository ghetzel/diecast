package diecast

import (
	"bytes"
	"io/ioutil"
	"net/http/httptest"
	"testing"

	"github.com/alecthomas/assert"
)

func TestRendererConfigWithResponse(t *testing.T) {
	var a RendererConfig

	assert.Nil(t, a.request)
	assert.Nil(t, a.data)

	var b = newRenderConfigFromRequest(
		httptest.NewRequest(`GET`, `/`, nil),
		ioutil.NopCloser(bytes.NewBufferString(`hello`)),
	)

	assert.NotNil(t, b.request)
	assert.NotNil(t, b.data)
}
