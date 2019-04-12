package diecast

import (
	"fmt"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func musturl(in string) *url.URL {
	if u, err := url.Parse(in); err == nil {
		return u
	} else {
		panic(fmt.Sprintf("invalid url: %v", err))
	}
}

func TestAuthenticatorConfigs(t *testing.T) {
	assert := require.New(t)

	auth0 := AuthenticatorConfig{
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

	auth1 := AuthenticatorConfig{
		Name: `primary`,
		Type: `always`,
		Except: []string{
			`/logout`,
			`*/assets/*`,
		},
	}

	auths := AuthenticatorConfigs{auth0, auth1}

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

// func TestBasicAuthenticator(t *testing.T) {
// 	assert := require.New(t)
// 	auth, err := NewBasicAuthenticator(&AuthenticatorConfig{})
// }
