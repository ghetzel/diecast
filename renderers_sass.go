//go:build cgo
// +build cgo

package diecast

// #cgo CFLAGS: -Wno-implicit-function-declaration
// #cgo CPPFLAGS: -I/usr/include -I/usr/local/include
// #cgo LDFLAGS: -L/usr/lib -L/usr/local/lib -lsass
// #include <sass/context.h>
// #include "renderers_sass.h"
import "C"

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"sync"
	"unsafe"
)

var SassIndentString = `    `
var callbackMap sync.Map

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

func (self *SassRenderer) SetServer(server *Server) {
	self.server = server
}

func (self *SassRenderer) Render(w http.ResponseWriter, req *http.Request, options RenderOptions) error {
	defer options.Input.Close()

	if data, err := ioutil.ReadAll(options.Input); err == nil {
		// setup Sass_Data_Context with the file contents we've been given
		var dctx = C.sass_make_data_context(C.CString(string(data)))
		defer C.sass_delete_data_context(dctx)

		// get the Sass_Context from said Sass_Data_Context
		var ctx = C.sass_data_context_get_context(dctx)

		// get Sass_Options
		var opt = C.sass_data_context_get_options(dctx)

		// set compile options
		C.sass_option_set_precision(opt, C.int(10))
		C.sass_option_set_source_comments(opt, C.bool(false))
		C.sass_option_set_indent(opt, C.CString(SassIndentString))

		// C.SASS_STYLE_NESTED
		// C.SASS_STYLE_EXPANDED
		// C.SASS_STYLE_COMPACT
		// C.SASS_STYLE_COMPRESSED
		C.sass_option_set_output_style(opt, C.SASS_STYLE_EXPANDED)

		var implist = C.sass_make_importer_list(C.ulong(1))
		var importer = C.sass_make_importer((C.Sass_Importer_Fn)(C.diecast_sass_importer), C.double(0), nil)
		var cookie = C.sass_importer_get_cookie(importer)
		callbackMap.Store(cookie, self)

		C.sass_importer_set_list_entry(implist, C.ulong(0), importer)
		C.sass_option_set_c_importers(opt, implist)

		// write options back to the data context
		C.sass_data_context_set_options(dctx, opt)

		if status := int(C.sass_compile_data_context(dctx)); status == 0 {
			var output = C.GoString(C.sass_context_get_output_string(ctx))

			w.Header().Set(`Content-Type`, `text/css; charset=utf-8`)

			if fn := self.prewrite; fn != nil {
				fn(req)
			}

			_, err := w.Write([]byte(output))
			return err
		} else {
			var err = errors.New(C.GoString(C.sass_context_get_error_message(ctx)))
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

//export go_retrievePath
func go_retrievePath(cookie unsafe.Pointer, url *C.char, output **C.char) C.int {
	var mesg string
	var code int

	if path := C.GoString(url); path != `` {
		if v, ok := callbackMap.Load(cookie); ok {
			if renderer, ok := v.(*SassRenderer); ok {
				var candidates = []string{path}

				// if the requested import path does not have a file extension, in addition to the
				// path given, also try with the .scss and .css extensions.
				if filepath.Ext(path) == `` {
					candidates = append(candidates, fmt.Sprintf("%s.scss", path))
					candidates = append(candidates, fmt.Sprintf("%s.css", path))
				}

				// try all candidate paths, first one wins
				for _, candidate := range candidates {
					if file, err := renderer.server.fs.Open(candidate); err == nil {
						defer file.Close()

						if data, err := ioutil.ReadAll(file); err == nil {
							mesg = string(data)
							code = len(data)
							break
						} else {
							mesg = fmt.Sprintf("Cannot read %v: %v", candidate, err)
							code = -5
						}
					} else {
						mesg = fmt.Sprintf("Cannot open %v: %v", candidate, err)
						code = -4
					}
				}
			} else {
				mesg = fmt.Sprintf("invalid callback mapping: expected SassRenderer, got %T", v)
				code = -3
			}
		} else {
			mesg = fmt.Sprintf("invalid callback mapping")
			code = -2
		}
	} else {
		mesg = fmt.Sprintf("no path specified")
		code = -1
	}

	*output = C.CString(mesg)
	return C.int(code)
}
