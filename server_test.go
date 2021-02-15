package diecast

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServerServeHTTP(t *testing.T) {
	var server interface{} = new(Server)
	var w = httptest.NewRecorder()
	var req = httptest.NewRequest(`GET`, `/`, nil)

	// ensure that we do, in fact, implement http.Handler
	server.(http.Handler).ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Equal(t, `Greetings.`, w.Body.String())
}
