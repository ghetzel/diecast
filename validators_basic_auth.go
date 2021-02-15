package diecast

import "net/http"

type BasicAuthValidator struct{}

func (self *BasicAuthValidator) Validate(cfg *ValidatorConfig, req *http.Request) error {
	return ErrNotImplemented
}
