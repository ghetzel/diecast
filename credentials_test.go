package diecast

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ghetzel/go-stockutil/typeutil"
	"github.com/stretchr/testify/assert"
)

func basicAuthHeader(username string, password string) string {
	return `Basic ` + base64.StdEncoding.EncodeToString([]byte(username+`:`+password))
}

func TestStaticCredentialProvider(t *testing.T) {
	var server Server

	server.VFS.Overrides = map[string]*File{
		`/index.html`: {
			Data: `Greetings.`,
		},
	}

	var static = &StaticCredentialProvider{}
	assert.NoError(t, static.AddUserCleartext(`test`, `correct`))
	server.SetCredentialProvider(static)
	server.Validators = []ValidatorConfig{
		{Type: `basic`},
	}

	// -------------------------------------------------------------------------------------------------------------------
	// missing credentials check
	var w = httptest.NewRecorder()
	var res, err = server.SimulateRequestWithRecorder(w, `GET`, `/`, nil, nil, nil)
	assert.ErrorContains(t, err, `HTTP 401: Unauthorized`)
	assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// -------------------------------------------------------------------------------------------------------------------
	// invalid credentials check
	w = httptest.NewRecorder()
	res, err = server.SimulateRequestWithRecorder(w, `GET`, `/`, nil, nil, map[string]any{
		`Authorization`: basicAuthHeader(`test`, `incorrect`),
	})
	assert.ErrorContains(t, err, `HTTP 401: Unauthorized`)
	assert.Equal(t, http.StatusUnauthorized, res.StatusCode)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// -------------------------------------------------------------------------------------------------------------------
	// valid credentials check
	w = httptest.NewRecorder()
	res, err = server.SimulateRequestWithRecorder(w, `GET`, `/`, nil, nil, map[string]any{
		`Authorization`: basicAuthHeader(`test`, `correct`),
	})
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, `Greetings.`, typeutil.String(res.Body))
}
