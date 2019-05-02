package diecast

// #cgo CPPFLAGS: -DUSE_LIBSASS -I/usr/include/sass -I/usr/local/include/sass
// #cgo LDFLAGS: -lsass
// #include <sass/context.h>
import "C"

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/ghetzel/go-stockutil/log"
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

	if data, err := ioutil.ReadAll(options.Input); err == nil {
		version := C.GoString(C.libsass_version())
		log.Debugf("%T: libsass v%v", self, version)

		// setup Sass_Data_Context with the file contents we've been given
		dctx := C.sass_make_data_context(C.CString(string(data)))
		defer C.sass_delete_data_context(dctx)

		// get the Sass_Context from said Sass_Data_Context
		ctx := C.sass_data_context_get_context(dctx)

		// get Sass_Options
		opt := C.sass_data_context_get_options(dctx)

		// set compile options
		C.sass_option_set_precision(opt, C.int(10))
		C.sass_option_set_source_comments(opt, C.bool(true))

		// write options back to the data context
		C.sass_data_context_set_options(dctx, opt)

		if status := int(C.sass_compile_data_context(dctx)); status == 0 {
			output := C.GoString(C.sass_context_get_output_string(ctx))

			w.Header().Set(`Content-Type`, `text/css; charset=utf-8`)

			if fn := self.prewrite; fn != nil {
				fn(req)
			}

			_, err := w.Write([]byte(output))
			return err
		} else {
			err := errors.New(C.GoString(C.sass_context_get_error_message(ctx)))
			return fmt.Errorf("Cannot render Sass: %v (status: %d)", err, status)
		}
	} else {
		return fmt.Errorf("Cannot read render input: %v", err)
	}
	// importer := libsass.NewImportsWithResolver(func(url string, prev string) (string, string, bool) {
	// 	if file, err := self.server.fs.Open(url); err == nil {
	// 		defer file.Close()

	// 		if data, err := ioutil.ReadAll(file); err == nil {
	// 			return url, string(data), true
	// 		} else {
	// 			log.Warningf("SassImport[%s]: %v", url, err)
	// 		}
	// 	} else {
	// 		log.Warningf("SassImport[%s]: %v", url, err)
	// 	}

	// 	return ``, ``, false
	// })
}
