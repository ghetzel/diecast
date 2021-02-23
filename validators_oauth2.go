package diecast

type OAuth2Validator struct{}

func (self *OAuth2Validator) Validate(ctx *Context, cfg *ValidatorConfig) error {
	return NotImplemented(self)
}
