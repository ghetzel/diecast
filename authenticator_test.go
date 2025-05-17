package diecast

import (
	"encoding/base64"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/ghetzel/testify/require"
	htpasswd "github.com/tg123/go-htpasswd"
)

func TestAuthenticatorConfigs(t *testing.T) {
	var assert = require.New(t)

	var auth0 = AuthenticatorConfig{
		Name: `admin`,
		Type: `always`,
		Paths: []string{
			`/admin`,
			`/admin/*`,
		},
		Except: []string{
			`/admin/assets/*`,
		},
	}

	var auth1 = AuthenticatorConfig{
		Name: `primary`,
		Type: `always`,
		Except: []string{
			`/logout`,
			`*/assets/*`,
		},
	}

	var auths = AuthenticatorConfigs{auth0, auth1}

	auth, err := auths.Authenticator(httptest.NewRequest(`GET`, `/`, nil))
	assert.NoError(err)
	assert.Equal(`primary`, auth.Name())

	auth, err = auths.Authenticator(httptest.NewRequest(`GET`, `/admin`, nil))
	assert.NoError(err)
	assert.Equal(`admin`, auth.Name())

	auth, err = auths.Authenticator(httptest.NewRequest(`GET`, `/admin/assets/its/cool/yay.css`, nil))
	assert.NoError(err)
	assert.Nil(auth)
}

func TestBasicAuthenticator(t *testing.T) {
	var assert = require.New(t)
	auth, err := NewBasicAuthenticator(&AuthenticatorConfig{
		Options: map[string]any{
			`credentials`: map[string]any{
				`tester01`: `{SHA}u3/Rg4+2cdohm4CmQtP9Qq45HX0=`,
			},
		},
	})

	assert.NoError(err)

	htp, err := htpasswd.AcceptSha(`{SHA}u3/Rg4+2cdohm4CmQtP9Qq45HX0=`)
	assert.NoError(err)
	assert.NotNil(htp)
	assert.True(htp.MatchesPassword(`t3st`))

	var req = httptest.NewRequest(`GET`, `/`, nil)
	req.Header.Set(`Authorization`, `Basic `+base64.StdEncoding.EncodeToString(
		[]byte(url.UserPassword(`tester01`, `t3st`).String()),
	))

	assert.True(auth.Authenticate(
		httptest.NewRecorder(),
		req,
	))

	req = httptest.NewRequest(`GET`, `/`, nil)
	req.Header.Set(`Authorization`, `Basic `+base64.StdEncoding.EncodeToString(
		[]byte(url.UserPassword(`tester01`, `WRONGPASSWORD`).String()),
	))

	assert.False(auth.Authenticate(
		httptest.NewRecorder(),
		req,
	))

	req = httptest.NewRequest(`GET`, `/`, nil)
	req.Header.Set(`Authorization`, `Basic `+base64.StdEncoding.EncodeToString(
		[]byte(url.UserPassword(`wrongUser`, `t3st`).String()),
	))

	assert.False(auth.Authenticate(
		httptest.NewRecorder(),
		req,
	))
}
