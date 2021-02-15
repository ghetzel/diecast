package diecast

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServerServeHTTP(t *testing.T) {
	var server Server
	var w = httptest.NewRecorder()
	var req *http.Request

	server.Paths.IndexFilename = `testing.html`

	server.VFS.Overrides = map[string]*File{
		`/testing.html`: {
			Data: `Greetings.`,
		},
		`/test.json`: {
			Data: map[string]interface{}{
				`hello`: `there`,
			},
		},
	}

	// validate the exposure and configurability of IndexFilename
	req = httptest.NewRequest(`GET`, `/`, nil)
	server.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, `Greetings.`, w.Body.String())

	// validate automatic encoding of complex types
	w = httptest.NewRecorder()
	req = httptest.NewRequest(`GET`, `/test.json`, nil)
	server.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "{\n  \"hello\": \"there\"\n}", w.Body.String())
}

func TestServerWriteResponse(t *testing.T) {
	var server Server
	var w = httptest.NewRecorder()
	var req = httptest.NewRequest(`GET`, `/`, nil)

	// -------------------------------------------------------------------------------------------------------------------
	server.writeResponse(w, req, nil)
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, ``, w.Body.String())

	// -------------------------------------------------------------------------------------------------------------------
	w = httptest.NewRecorder()
	server.writeResponse(w, req, fmt.Errorf("test"))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, `test`, w.Body.String())

	// -------------------------------------------------------------------------------------------------------------------
	w = httptest.NewRecorder()
	server.writeResponse(w, req, fmt.Errorf("test"), http.StatusConflict)
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Equal(t, `test`, w.Body.String())

	// -------------------------------------------------------------------------------------------------------------------
	w = httptest.NewRecorder()
	server.writeResponse(w, req, fmt.Errorf("test"), http.StatusOK)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, `test`, w.Body.String())

	// -------------------------------------------------------------------------------------------------------------------
	w = httptest.NewRecorder()
	server.writeResponse(w, req, map[string]interface{}{
		`hello`: `there`,
	})

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, `application/json`, w.HeaderMap.Get(`Content-Type`))
	assert.Equal(t, "{\n  \"hello\": \"there\"\n}", w.Body.String())

	// -------------------------------------------------------------------------------------------------------------------
	w = httptest.NewRecorder()
	server.writeResponse(w, req, []map[string]interface{}{
		{
			`hello`: `there`,
		},
	})

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, `application/json`, w.HeaderMap.Get(`Content-Type`))
	assert.Equal(t, "[\n  {\n    \"hello\": \"there\"\n  }\n]", w.Body.String())
}
