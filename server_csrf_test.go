package diecast

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ghetzel/testify/require"
)

func TestCsrfRequest(t *testing.T) {
	var assert = require.New(t)

	var csrf = new(CSRF)

	var req = httptest.NewRequest(`GET`, `/`, nil)
	var w = httptest.NewRecorder()
	assert.True(csrf.Handle(w, req))
	assert.Empty(csrftoken(req))
	assert.Equal(http.StatusOK, w.Code)
	assert.Equal(``, w.HeaderMap.Get(DefaultCsrfHeaderName))

}

func TestCsrfRequestEnabled(t *testing.T) {
	var assert = require.New(t)
	var csrf = &CSRF{
		Enable: true,
	}

	var req = httptest.NewRequest(`GET`, `/`, nil)
	var w = httptest.NewRecorder()
	assert.True(csrf.Handle(w, req))
	assert.Equal(http.StatusOK, w.Code)
	assert.Equal(
		csrftoken(req),
		w.HeaderMap.Get(DefaultCsrfHeaderName),
	)

}

func TestCsrfPostInvalid(t *testing.T) {
	var assert = require.New(t)
	var csrf = &CSRF{
		Enable: true,
	}

	// try a bare POST (no token)
	// ----------------------------------------------------------------------
	var req = httptest.NewRequest(`POST`, `/thing`, nil)
	var w = httptest.NewRecorder()
	assert.False(csrf.Handle(w, req))
	assert.Equal(http.StatusBadRequest, w.Code)
}

func TestCsrfPostInvalidNoCookie(t *testing.T) {
	var assert = require.New(t)
	var csrf = &CSRF{
		Enable: true,
	}

	// now add the token (header, no cookie)
	// ----------------------------------------------------------------------
	var req = httptest.NewRequest(`POST`, `/thing`, nil)
	req.Header.Set(DefaultCsrfHeaderName, `abc123`)

	var w = httptest.NewRecorder()
	assert.False(csrf.Handle(w, req))
	assert.Equal(http.StatusBadRequest, w.Code)

}

func TestCsrfPostValid(t *testing.T) {
	var assert = require.New(t)
	var csrf = &CSRF{
		Enable: true,
	}

	// now add the token (header, cookie w/ same value)
	// ----------------------------------------------------------------------
	var req = httptest.NewRequest(`POST`, `/thing`, nil)
	req.Header.Set(DefaultCsrfHeaderName, `abc123`)
	req.AddCookie(&http.Cookie{
		Name:  DefaultCsrfCookieName,
		Value: `abc123`,
	})

	var w = httptest.NewRecorder()
	assert.True(csrf.Handle(w, req))
	assert.Equal(http.StatusOK, w.Code)

}

func TestCsrfInvalidWrongCookie(t *testing.T) {
	var assert = require.New(t)
	var csrf = &CSRF{
		Enable: true,
	}

	// now add the token (header, cookie w/ different value)
	// ----------------------------------------------------------------------
	var req = httptest.NewRequest(`POST`, `/thing`, nil)
	req.Header.Set(DefaultCsrfHeaderName, `abc123`)
	req.AddCookie(&http.Cookie{
		Name:  DefaultCsrfCookieName,
		Value: `potato`,
	})

	var w = httptest.NewRecorder()
	assert.False(csrf.Handle(w, req))
	assert.Equal(http.StatusBadRequest, w.Code)
}

func TestCsrfPostValidRequestBodyIntact(t *testing.T) {
	var assert = require.New(t)
	var csrf = &CSRF{
		Enable: true,
	}

	var body = bytes.NewBufferString("everything is very okay")

	// now add the token (header, cookie w/ same value)
	// ----------------------------------------------------------------------
	var req = httptest.NewRequest(`POST`, `/thing`, body)
	req.Header.Set(DefaultCsrfHeaderName, `abc123`)
	req.AddCookie(&http.Cookie{
		Name:  DefaultCsrfCookieName,
		Value: `abc123`,
	})

	var w = httptest.NewRecorder()
	assert.True(csrf.Handle(w, req))
	assert.Equal(http.StatusOK, w.Code)

	reqbody, err := ioutil.ReadAll(req.Body)
	assert.NoError(err)

	// the request body should still contain everything it had
	assert.Equal(`everything is very okay`, string(reqbody))

	// utilizing the "abc123" token should have forced a new token
	assert.NotEqual(`abc123`, csrftoken(req))
	assert.Equal(csrftoken(req), w.HeaderMap.Get(DefaultCsrfHeaderName))
}
