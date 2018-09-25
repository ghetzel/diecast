package diecast

import (
	"fmt"
	"net/http"

	"github.com/gobwas/glob"
)

type Authenticator interface {
	Authenticate(http.ResponseWriter, *http.Request) bool
}

type AuthenticatorConfig struct {
	Type    string                 `json:"type"`
	Paths   []string               `json:"paths"`
	Options map[string]interface{} `json:"options"`
	globs   []glob.Glob
}

type AuthenticatorConfigs []AuthenticatorConfig

func (self AuthenticatorConfigs) Authenticator(req *http.Request) (Authenticator, error) {
	for _, auth := range self {
		if len(auth.Paths) != len(auth.globs) {
			auth.globs = nil

			for _, pattern := range auth.Paths {
				auth.globs = append(auth.globs, glob.MustCompile(pattern))
			}
		}

		if len(auth.globs) > 0 {
			for _, px := range auth.globs {
				if px.Match(req.URL.Path) {
					return returnAuthenticatorFor(&auth)
				}
			}
		} else {
			return returnAuthenticatorFor(&auth)
		}
	}

	return nil, nil
}

func returnAuthenticatorFor(auth *AuthenticatorConfig) (Authenticator, error) {
	var authenticator Authenticator
	var err error

	switch auth.Type {
	case `basic`:
		authenticator, err = NewBasicAuthenticator(auth.Options)
	default:
		err = fmt.Errorf("unrecognized authenticator type %q", auth.Type)
	}

	return authenticator, err
}
