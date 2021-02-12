package diecast

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/ghetzel/go-stockutil/fileutil"
)

type OOXMLRenderer struct {
	server   *Server
	prewrite PrewriteFunc
}

func (self *OOXMLRenderer) ShouldPrerender() bool {
	return false
}

func (self *OOXMLRenderer) SetServer(server *Server) {
	self.server = server
}

func (self *OOXMLRenderer) SetPrewriteFunc(fn PrewriteFunc) {
	self.prewrite = fn
}

func (self *OOXMLRenderer) Render(w http.ResponseWriter, req *http.Request, options RenderOptions) error {
	defer options.Input.Close()

	// 1. mimetype the input to verify its an OOXML format we support
	// -----------------------------------------------------------------------------------------------
	var inputType = fileutil.GetMimeType(options.Input)
	var ooxml bool

	switch inputType {
	case `application/vnd.openxmlformats-officedocument.wordprocessingml.document`:
		ooxml = true
	case `application/vnd.openxmlformats-officedocument.presentationml.presentation`:
		ooxml = true
	case `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`:
		ooxml = true
	}

	if ooxml {
		if inputData, err := ioutil.ReadAll(options.Input); err == nil {
			var input = bytes.NewReader(inputData)
			var output = zip.NewWriter(w)

			// its all gas and no brakes from here: files are read from the zip, rendered, and written to the response in one pipeline
			if inzip, err := zip.NewReader(input, int64(input.Len())); err == nil {
				w.Header().Set(`Content-Type`, inputType)

				for _, inpart := range inzip.File {
					if outpart, err := output.CreateHeader(&inpart.FileHeader); err == nil {
						if src, err := inpart.Open(); err == nil {
							return fmt.Errorf("NOT IMPLEMENTED: %v %v", src, outpart)
						} else {
							return fmt.Errorf("bad part %q: %v", inpart.Name, err)
						}
					} else {
						return err
					}
				}

				return nil
			} else {
				return fmt.Errorf("bad archive: %v", err)
			}
		} else {
			return err
		}
	} else {
		return fmt.Errorf("unsupported input type %q", inputType)
	}
}
