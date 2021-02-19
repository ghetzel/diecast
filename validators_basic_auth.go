package diecast

type BasicAuthValidator struct{}

func (self *BasicAuthValidator) Validate(cfg *ValidatorConfig) error {
	return NotImplemented(self)
}
