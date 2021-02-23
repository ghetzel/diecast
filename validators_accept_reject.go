package diecast

type AcceptRejectValidator struct {
	// Addresses []string `yaml:"addresses"`
}

func (self AcceptRejectValidator) Validate(ctx *Context, cfg *ValidatorConfig) error {
	return NotImplemented(self)
}
