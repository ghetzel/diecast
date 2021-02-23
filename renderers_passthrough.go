package diecast

import (
	"io"
	"net/http"
)

type PassthroughRenderer struct{}

func Passthrough(ctx *Context, input http.File, _ *RendererConfig) error {
	return new(PassthroughRenderer).Render(ctx, input, nil)
}

func (self *PassthroughRenderer) Render(ctx *Context, input http.File, _ *RendererConfig) error {
	_, err := io.Copy(ctx, input)
	return err
}
