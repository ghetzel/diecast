package diecast

import (
	"io"
	"net/http"
)

type PassthroughRenderer struct{}

func Passthrough(w http.ResponseWriter, cfg *RendererConfig) error {
	return new(PassthroughRenderer).Render(w, cfg)
}

func (self *PassthroughRenderer) Render(w http.ResponseWriter, cfg *RendererConfig) error {
	_, err := io.Copy(w, cfg.Data)
	return err
}
