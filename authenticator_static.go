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

func (self *StaticAuthenticator) Name() string {
	if self.config != nil && self.config.Name != `` {
		return self.config.Name
	} else {
		return `StaticAuthenticator`
	}
}

func (self *StaticAuthenticator) IsCallback(_ *url.URL) bool {
	return false
}

func (self *StaticAuthenticator) Callback(w http.ResponseWriter, req *http.Request) {

}

func (self *StaticAuthenticator) Authenticate(w http.ResponseWriter, req *http.Request) bool {
	return self.allow
}
