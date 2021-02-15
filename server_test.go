package diecast

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServerServeHTTP(t *testing.T) {
	var server Server
	var w = httptest.NewRecorder()
	var req = httptest.NewRequest(`GET`, `/`, nil)

	server.vfs.OverridePath(`/index.html`, `Greetings.`)

	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, `Greetings.`, w.Body.String())
}
