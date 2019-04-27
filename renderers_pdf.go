package diecast

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/ghetzel/go-stockutil/httputil"
	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-webfriend"
	"github.com/ghetzel/go-webfriend/browser"
	wfcore "github.com/ghetzel/go-webfriend/commands/core"
	wfpage "github.com/ghetzel/go-webfriend/commands/page"
)

type PdfRenderer struct {
	server   *Server
	prewrite PrewriteFunc
}

func (self *PdfRenderer) ShouldPrerender() bool {
	return false
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

	} else if www, err := browser.Start(); err == nil {
		defer www.Stop()
		var buffer bytes.Buffer

		subaddr := self.server.Address

		if strings.HasPrefix(subaddr, `:`) {
			subaddr = `127.0.0.1` + subaddr
		}

		// start a headless chromium-browser instance that we can interact with
		env := webfriend.NewEnvironment(www)

		// mangle the URL to be a strictly-localhost affair
		suburl, _ := url.Parse(req.URL.String())
		suburl.Scheme = `http`
		suburl.Host = subaddr
		subqs := suburl.Query()
		subqs.Set(`__subrender`, `true`)
		suburl.RawQuery = subqs.Encode()

		log.Debugf("Rendering %v as PDF", suburl)

		core := env.MustModule(`core`).(*wfcore.Commands)
		page := env.MustModule(`page`).(*wfpage.Commands)

		var timeout time.Duration

		if options.Timeout > 0 {
			timeout = options.Timeout
		} else {
			timeout = 60 * time.Second
		}

		// visit the URL
		if _, err := core.Go(suburl.String(), &wfcore.GoArgs{
			Timeout:                   timeout,
			RequireOriginatingRequest: false,
		}); err != nil {
			return err
		}

		// render the loaded page as a PDF
		if err := page.Pdf(&buffer, nil); err == nil {
			w.Header().Set(`Content-Type`, `application/pdf`)

			rewrittenFilename := strings.TrimSuffix(
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
	} else {
		log.Fatalf("could not generate PDF: %v", err)
		return err
	}
}
