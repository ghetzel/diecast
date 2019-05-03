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

	server := NewServer(`./examples/hello-world`)
	mounts := getTestMounts(assert)
	server.SetMounts(mounts)
	assert.NoError(server.Initialize())

	renderer := new(SassRenderer)
	renderer.server = server

	request := httptest.NewRequest(`GET`, `/css/for-sass.scss`, nil)
	recorder := httptest.NewRecorder()

	testsass := `.parent { td { color: red; } tr { color: blue }}; @import '/css/for-sass';`

	assert.NoError(renderer.Render(recorder, request, RenderOptions{
		Input: ioutil.NopCloser(bytes.NewBufferString(testsass)),
	}))

	res := recorder.Result()
	assert.NotNil(res)
	assert.Equal(`text/css; charset=utf-8`, res.Header.Get(`Content-Type`))
	assert.NotNil(res.Body)
	assert.Equal(".parent td {\n    color: red;\n}\n\n.parent tr {\n    color: blue;\n}\n\nh1 {\n    color: red;\n}\n", typeutil.String(res.Body))
	t.Logf("Test complete")
}
