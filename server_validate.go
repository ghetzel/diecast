package diecast

import (
	"fmt"
	"net/http"
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

// Validate the given request against all configured validators.  Will return nil if
// the request passes all matching validations or only fails on optional ones.
func (self *Server) ValidateAll(req *http.Request) error {
	if err := self.prep(); err != nil {
		return err
	}

	// check the request against each configured validator
	for _, vc := range self.Validators {
		if vc.Type != `` {
			if vc.ShouldValidateRequest(req) {
				if validator, ok := validators[vc.Type]; ok {
					if err := validator.Validate(&vc, req); err != nil {
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
