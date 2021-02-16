package diecast

import (
	"fmt"
	"net/http"

	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/typeutil"
)

var validators = make(map[string]Validator)

func init() {
	RegisterValidator(`accept`, new(AcceptRejectValidator))
	RegisterValidator(`reject`, new(AcceptRejectValidator))
	RegisterValidator(`basic`, new(BasicAuthValidator))
	RegisterValidator(`oauth2`, new(OAuth2Validator))
}

func RegisterValidator(name string, validator Validator) {
	validators[name] = validator
}

type ValidatorConfig struct {
	Type     string                 `yaml:"type"`
	Options  map[string]interface{} `yaml:"options"`
	Only     interface{}            `yaml:"only"`
	Except   interface{}            `yaml:"except"`
	Methods  interface{}            `yaml:"methods"`
	Optional bool                   `yaml:"optional"`
	Request  *http.Request          `yaml:"-"`
}

// Return whether the given request is eligible for validation under normal circumstances.
func (self *ValidatorConfig) ShouldApplyTo(req *http.Request) bool {
	return ShouldApplyTo(req, self.Except, self.Only, self.Methods)
}

// Return a typeutil.Variant containing the value at the named option key, or a fallback value.
func (self *ValidatorConfig) Option(name string, fallbacks ...interface{}) typeutil.Variant {
	return maputil.M(self.Options).Get(name, fallbacks...)
}

func (self ValidatorConfig) WithRequest(req *http.Request) *ValidatorConfig {
	var cfg = self
	cfg.Request = req
	return &cfg
}

// =====================================================================================================================

// Validate the given request against all configured validators.  Will return nil if
// the request passes all matching validations or only fails on optional ones.
func (self *Server) ValidateRequest(req *http.Request) error {
	if err := self.prep(); err != nil {
		return err
	}

	// check the request against each configured validator
	for _, vc := range self.Validators {
		if vc.Type != `` {
			if vc.ShouldApplyTo(req) {
				if validator, ok := validators[vc.Type]; ok {
					if err := validator.Validate(vc.WithRequest(req)); err != nil {
						if !vc.Optional {
							return fmt.Errorf("failed on %q validator: %v", vc.Type, err)
						}
					}
				}
			}
		} else {
			return fmt.Errorf("unrecognized validator type %q", vc.Type)
		}
	}

	return nil
}
