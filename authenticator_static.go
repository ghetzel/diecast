package diecast

import (
	"net/http"
	"net/url"
)

type StaticAuthenticator struct {
	config *AuthenticatorConfig
	allow  bool
}

func NewStaticAuthenticator(config *AuthenticatorConfig, allow bool) (*StaticAuthenticator, error) {
	return &StaticAuthenticator{
		config: config,
		allow:  allow,
	}, nil
}

func (auth *StaticAuthenticator) Name() string {
	if auth.config != nil && auth.config.Name != `` {
		return auth.config.Name
	} else {
		return `StaticAuthenticator`
	}
}

func (auth *StaticAuthenticator) IsCallback(_ *url.URL) bool {
	return false
}

func (auth *StaticAuthenticator) Callback(w http.ResponseWriter, req *http.Request) {

}

func (auth *StaticAuthenticator) Authenticate(w http.ResponseWriter, req *http.Request) bool {
	return auth.allow
}
