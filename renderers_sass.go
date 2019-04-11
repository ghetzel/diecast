// +build !nocgo

package diecast

import (
	"io/ioutil"
	"net/http"

	"github.com/ghetzel/go-stockutil/log"
	"github.com/wellington/go-libsass"
)

type SassRenderer struct {
	server   *Server
	prewrite PrewriteFunc
}

func (self *SassRenderer) ShouldPrerender() bool {
	return true
}

func (self *SassRenderer) SetPrewriteFunc(fn PrewriteFunc) {
	self.prewrite = fn
}

func (self *SassRenderer) Render(w http.ResponseWriter, req *http.Request, options RenderOptions) error {
	defer options.Input.Close()

	importer := libsass.NewImportsWithResolver(func(url string, prev string) (string, string, bool) {
		if file, err := self.server.fs.Open(url); err == nil {
			defer file.Close()

			if data, err := ioutil.ReadAll(file); err == nil {
				return url, string(data), true
			} else {
				log.Warningf("SassImport[%s]: %v", url, err)
			}
		} else {
			log.Warningf("SassImport[%s]: %v", url, err)
		}

		return ``, ``, false
	})

	importer.Init()

	if sass, err := libsass.New(
		w,
		options.Input,
		libsass.OutputStyle(libsass.EXPANDED_STYLE),
		libsass.ImportsOption(importer),
	); err == nil {
		w.Header().Set(`Content-Type`, `text/css; charset=utf-8`)

		if fn := self.prewrite; fn != nil {
			fn(req)
		}

		return sass.Run()
	} else {
		return err
	}
}
