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
	Except   []string               `json:"except"`
	Options map[string]interface{} `json:"options"`
	globs   []glob.Glob
	exceptGlobs []glob.Glob
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

		if len(auth.Except) != len(auth.exceptGlobs) {
			auth.exceptGlobs = nil

			for _, pattern := range auth.Except {
				auth.exceptGlobs = append(auth.exceptGlobs, glob.MustCompile(pattern))
			}
		}

		if self.isUrlMatch(auth, req.URL) {
			return returnAuthenticatorFor(&auth)
		}
	}

	return nil, nil
}

func (self AuthenticatorConfigs) isUrlMatch(auth Authenticator, u *url.URL) bool {
	var match bool

	// determine if any of our paths match the request path
	for _, px := range auth.globs {
		if px.Match(u.Path) {
			match = true
			break
		}
	}

	// no matches? then except wouldn't do anything anyway. return false now
	if !match {
		return false
	}

	// we have at least one match, make sure we don't run afould of any excepts
	for _, xx := range auth.exceptGlobs {
		if xx.Match(u.Path) {
			return false
		}
	}

	// we got here: this URL matches the given Authenticator
	return true
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
