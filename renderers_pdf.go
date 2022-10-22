package diecast

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/ghetzel/go-stockutil/httputil"
	"github.com/ghetzel/go-stockutil/log"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

type PdfRenderer struct {
	server   *Server
	prewrite PrewriteFunc
}

func (self *PdfRenderer) ShouldPrerender() bool {
	return false
}

func (self *PdfRenderer) SetServer(server *Server) {
	self.server = server
}

func (self *PdfRenderer) SetPrewriteFunc(fn PrewriteFunc) {
	self.prewrite = fn
}

func (self *PdfRenderer) Render(w http.ResponseWriter, req *http.Request, options RenderOptions) error {
	defer options.Input.Close()

	if httputil.QBool(req, `__subrender`) {
		if fn := self.prewrite; fn != nil {
			fn(req)
		}

		_, err := io.Copy(w, options.Input)
		return err
	} else {
		var buffer bytes.Buffer
		var browser = rod.New()

		if err := browser.Connect(); err != nil {
			return err
		}

		defer browser.Close()

		var subaddr = self.server.Address

		if strings.HasPrefix(subaddr, `:`) {
			subaddr = `127.0.0.1` + subaddr
		}

		// mangle the URL to be a strictly-localhost affair
		suburl, _ := url.Parse(req.URL.String())
		suburl.Scheme = `http`
		suburl.Host = subaddr
		var subqs = suburl.Query()
		subqs.Set(`__subrender`, `true`)
		suburl.RawQuery = subqs.Encode()

		log.Debugf("Rendering %v as PDF", suburl)

		if data, err := browser.MustPage(suburl.String()).Screenshot(true, &proto.PageCaptureScreenshot{
			Format: proto.PageCaptureScreenshotFormatPng,
		}); err == nil {
			buffer.Write(data)
			w.Header().Set(`Content-Type`, `application/pdf`)

			var rewrittenFilename = strings.TrimSuffix(
				filepath.Base(options.RequestedPath),
				filepath.Ext(options.RequestedPath),
			) + `.pdf`

			w.Header().Set(`Content-Disposition`, fmt.Sprintf("inline; filename=%q", rewrittenFilename))

			if fn := self.prewrite; fn != nil {
				fn(req)
			}

			_, err := io.Copy(w, &buffer)

			return err
		} else {
			return err
		}
	}
}
