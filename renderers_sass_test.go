package diecast

import (
	"bytes"
	"io/ioutil"
	"net/http/httptest"
	"testing"

	"github.com/ghetzel/go-stockutil/typeutil"
	"github.com/stretchr/testify/require"
)

func TestSassRenderer(t *testing.T) {
	assert := require.New(t)
	renderer := new(SassRenderer)
	request := httptest.NewRequest(`GET`, `/test.scss`, nil)
	recorder := httptest.NewRecorder()

	testsass := `.parent { td { color: red; } tr { color: blue }}`

	assert.NoError(renderer.Render(recorder, request, RenderOptions{
		Input: ioutil.NopCloser(bytes.NewBufferString(testsass)),
	}))

	res := recorder.Result()
	assert.NotNil(res)
	assert.Equal(`text/css; charset=utf-8`, res.Header.Get(`Content-Type`))
	assert.NotNil(res.Body)
	assert.Equal(`nope`, typeutil.String(res.Body))
}
