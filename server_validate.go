package diecast

import (
	"fmt"
	"net/http"

	"github.com/ghetzel/go-stockutil/sliceutil"
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
}

// Return whether the given request is eligible for validation under normal circumstances.
func (self ValidatorConfig) ShouldValidateRequest(req *http.Request) bool {
	for _, except := range sliceutil.UniqueStrings(self.Except) {
		if IsGlobMatch(req.URL.Path, except) {
			return false
		}
	}

	// if there are "only" paths, then we may still match something.
	// if not, then we didn't match an "except" path, and therefore should validate
	if onlys := sliceutil.UniqueStrings(self.Only); len(onlys) > 0 {
		for _, only := range onlys {
			if IsGlobMatch(req.URL.Path, only) {
				return true
			}
		}

		return false
	} else {
		return true
	}
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
