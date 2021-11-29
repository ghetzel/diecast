package diecast

import (
	"io/fs"
	"net/http"
)

// Used for exposing a desired status code when writing the response to an HTTP request.
type Codeable interface {
	Code() int
}

// Used for validating that an HTTP request may proceed to the Retrieve stage.
type Validator interface {
	Validate(*Context, *ValidatorConfig) error
}

// Implements a function that will retrieve the appropriate data for a given request.
type Retriever interface {
	Retrieve(*Context) (http.File, error)
}

// Takes a readable http.File, performs any desired conversion, and writes the result out to the given http.ResponseWriter.
type Renderer interface {
	Render(*Context, fs.File, *RendererConfig) error
}
