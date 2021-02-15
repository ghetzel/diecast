package diecast

import (
	"net/http"
	"path/filepath"
)

// Used for exposing a desired status code when writing the response to an HTTP request.
type Codeable interface {
	Code() int
}

type ValidatorConfig struct {
	Type     string                 `yaml:"type"`
	Options  map[string]interface{} `yaml:"options"`
	Only     []string               `yaml:"only"`
	Except   []string               `yaml:"except"`
	Methods  []string               `yaml:"methods"`
	Optional bool                   `yaml:"optional"`
}

func (self ValidatorConfig) ShouldValidateRequest(req *http.Request) bool {
	for _, except := range self.Except {
		if ok, err := filepath.Match(except, req.URL.Path); err == nil && ok {
			return false
		}
	}

	// if there are "only" paths, then we may still match something.
	// if not, then we didn't match an "except" path, and therefore should validate
	if len(self.Only) > 0 {
		for _, only := range self.Only {
			if ok, err := filepath.Match(only, req.URL.Path); err == nil && ok {
				return true
			}
		}

		return false
	} else {
		return true
	}
}

// Used for validating that an HTTP request may proceed to the Retrieve stage.
type Validator interface {
	Validate(*ValidatorConfig, *http.Request) error
}

// Implements a function that will retrieve the appropriate data for a given request.
type Retriever interface {
	Retrieve(*http.Request) (http.File, error)
}

// Takes a readable http.File, performs any desired conversion, and writes the result out to the given http.ResponseWriter.
type Renderer interface {
	Render(http.ResponseWriter, *http.Request, http.File) error
}
