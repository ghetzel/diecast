package diecast

type BasicAuthValidator struct{}

func (self *BasicAuthValidator) Validate(ctx *Context, cfg *ValidatorConfig) error {
	return NotImplemented(self)
}
