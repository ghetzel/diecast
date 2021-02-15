package diecast

import "net/http"

type OAuth2Validator struct{}

func (self *OAuth2Validator) Validate(cfg *ValidatorConfig, req *http.Request) error {
	return ErrNotImplemented
}
