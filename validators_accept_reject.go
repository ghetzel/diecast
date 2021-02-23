package diecast

type AcceptRejectValidator struct{}

func (self *AcceptRejectValidator) Validate(ctx *Context, cfg *ValidatorConfig) error {
	return NotImplemented(self)
}
