package diecast

import (
	"io"
	"io/fs"
)

type PassthroughRenderer struct{}

func Passthrough(ctx *Context, input fs.File, _ *RendererConfig) error {
	return new(PassthroughRenderer).Render(ctx, input, nil)
}

func (self *PassthroughRenderer) Render(ctx *Context, input fs.File, _ *RendererConfig) error {
	_, err := io.Copy(ctx, input)
	return err
}
