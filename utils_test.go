package diecast

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMockHttpFile(t *testing.T) {
	var file *mockFile
	var err error

	// file, err = newMockFile(`test.txt`, nil)
	// assert.Equal(t, ErrNotFound, err)

	file, err = newMockFile(`test.txt`, `HELLO`)
	assert.NoError(t, err)
	assert.Equal(t, `HELLO`, file.String())

	file, err = newMockFile(`test.txt`, []byte(`HELLO`))
	assert.NoError(t, err)
	assert.Equal(t, `HELLO`, file.String())

	file, err = newMockFile(`test.txt`, bytes.NewBufferString(`HELLO`))
	assert.NoError(t, err)
	assert.Equal(t, `HELLO`, file.String())

	file, err = newMockFile(`test.txt`, errors.New(`HELLO`))
	assert.NoError(t, err)
	assert.Equal(t, `HELLO`, file.String())
}

func TestIsGlobMatch(t *testing.T) {
	assert.True(t, IsGlobMatch(`/hello/there.html`, `/hello/there.html`))
	assert.True(t, IsGlobMatch(`/hello/there.html`, `/hello/*.html`))
	assert.True(t, IsGlobMatch(`/hello/there.html`, `*.html`))

	assert.False(t, IsGlobMatch(`/hello/there.html`, `/hello/*.yaml`))
	assert.False(t, IsGlobMatch(`/hello/there.html`, `^/*.html`))

	assert.False(t, IsGlobMatch(`/hello/there.html`, `[0-`))
}

type shouldApplyToFunc = func(*http.Request, interface{}, interface{}, interface{}) bool

func TestShouldApplyTo(t *testing.T) {
	for _, satfn := range []shouldApplyToFunc{
		ShouldApplyTo,
		func(req *http.Request, except interface{}, only interface{}, methods interface{}) bool {
			var c = new(ValidatorConfig)

			c.Except = except
			c.Only = only
			c.Methods = methods

			return c.ShouldApplyTo(req)
		},
		func(req *http.Request, except interface{}, only interface{}, methods interface{}) bool {
			var c = new(RendererConfig)

			c.Except = except
			c.Only = only
			c.Methods = methods

			return c.ShouldApplyTo(req)
		},
	} {
		var req = httptest.NewRequest(`GET`, `/hello/there.html`, nil)

		assert.True(t, satfn(req, nil, nil, nil))
		assert.True(t, satfn(req, nil, `/hello/there.html`, nil))

		req = httptest.NewRequest(`GET`, `/other.html`, nil)
		assert.False(t, satfn(req, nil, `/hello/there.html`, nil))

		var allButYamlAndJson = func(r *http.Request) bool {
			return satfn(r, []string{
				`*.yaml`,
				`*.json`,
			}, nil, nil)
		}

		req = httptest.NewRequest(`GET`, `/file.html`, nil)
		assert.True(t, allButYamlAndJson(req))

		req = httptest.NewRequest(`GET`, `/file.yaml`, nil)
		assert.False(t, allButYamlAndJson(req))

		req = httptest.NewRequest(`GET`, `/file.json`, nil)
		assert.False(t, allButYamlAndJson(req))

		var noDeletes = func(r *http.Request) bool {
			return satfn(r, nil, nil, []string{
				`GET`,
				`POST`,
				`PUT`,
			})
		}

		req = httptest.NewRequest(`GET`, `/file.html`, nil)
		assert.True(t, noDeletes(req))

		req = httptest.NewRequest(`POST`, `/file.yaml`, nil)
		assert.True(t, noDeletes(req))

		req = httptest.NewRequest(`DELETE`, `/file.json`, nil)
		assert.False(t, noDeletes(req))
	}
}
