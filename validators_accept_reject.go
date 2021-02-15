package diecast

import "net/http"

type AcceptRejectValidator struct{}

func (self *AcceptRejectValidator) Validate(cfg *ValidatorConfig, req *http.Request) error {
	return ErrNotImplemented
}
