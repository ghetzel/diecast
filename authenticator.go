package diecast

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/ghetzel/go-stockutil/typeutil"
	"github.com/gobwas/glob"
)

type Authenticator interface {
	Authenticate(http.ResponseWriter, *http.Request) bool
	IsCallback(*url.URL) bool
	Callback(w http.ResponseWriter, req *http.Request)
	Name() string
}

type AuthenticatorConfig struct {
	Name         string                 `yaml:"name,omitempty" json:"name,omitempty"` // The name of the Authenticator
	Type         string                 `yaml:"type"           json:"type"`           // The type of Authenticator to create
	Paths        []string               `yaml:"paths"          json:"paths"`          // Which paths this Authenticator should apply to (defaults to all paths)
	Except       []string               `yaml:"except"         json:"except"`         // Specific paths this Authenticator should not cover (defaults to none)
	CallbackPath string                 `yaml:"callback"       json:"callback"`       // A secondary path this authenticator should redirect to (for multi-step Authenticators)
	Options      map[string]interface{} `yaml:"options"        json:"options"`        // Type-specific options
	globs        []glob.Glob
	exceptGlobs  []glob.Glob
}

func (self *AuthenticatorConfig) O(key string, fallback ...interface{}) typeutil.Variant {
	if len(self.Options) > 0 {
		if v, ok := self.Options[key]; ok {
			return typeutil.V(v)
		}
	}

	if len(fallback) > 0 {
		return typeutil.V(fallback[0])
	} else {
		return typeutil.V(nil)
	}
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

		if authIsUrlMatch(&auth, req.URL) {
			return returnAuthenticatorFor(&auth)
		}
	}

	return nil, nil
}

func authIsUrlMatch(auth *AuthenticatorConfig, u *url.URL) bool {
	var match bool

	// determine if any of our paths match the request path
	if len(auth.globs) > 0 {
		for _, px := range auth.globs {
			if px.Match(u.Path) {
				match = true
				break
			}
		}
	} else {
		match = true
	}

	// no matches? then except wouldn't do anything anyway. return false now
	if !match {
		return false
	}

	// we have at least one match, make sure we don't run afoul of any excepts
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
		authenticator, err = NewBasicAuthenticator(auth)
	case `oauth2`:
		authenticator, err = NewOauthAuthenticator(auth)
	case `shell`:
		authenticator, err = NewShellAuthenticator(auth)
	case `always`:
		authenticator, err = NewStaticAuthenticator(auth, true)
	case `never`:
		authenticator, err = NewStaticAuthenticator(auth, false)
	default:
		err = fmt.Errorf("unrecognized authenticator type %q", auth.Type)
	}

	return authenticator, err
}
