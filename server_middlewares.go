package diecast

import (
	"bytes"
	"crypto/x509"
	"fmt"
	"image"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"time"

	ico "github.com/biessek/golang-ico"
	"github.com/ghetzel/go-stockutil/executil"
	"github.com/ghetzel/go-stockutil/fileutil"
	"github.com/ghetzel/go-stockutil/httputil"
	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/typeutil"
	base58 "github.com/jbenet/go-base58"
	"github.com/urfave/negroni"
)

func (self *Server) setupServer() error {
	fileutil.InitMime()
	self.handler = negroni.New()

	// setup panic recovery handler
	self.handler.Use(negroni.NewRecovery())

	// setup request ID generation
	self.handler.UseFunc(func(w http.ResponseWriter, req *http.Request, next http.HandlerFunc) {
		defer next(w, req)
		requestId := base58.Encode(stringutil.UUID().Bytes())

		log.Debugf("[%s] %s", requestId, strings.Repeat(`-`, 69))
		log.Infof("[%s] %s %s", requestId, req.Method, req.URL.Path)
		log.Debugf("[%s] middleware: request id", requestId)

		httputil.RequestSetValue(req, ContextRequestKey, requestId)
		httputil.RequestSetValue(req, ContextResponseKey, w)

		w.Header().Set(`X-Diecast-Request-ID`, requestId)

		// setup request tracing info
		startRequestTimer(req)
	})

	// handle request dumper
	self.handler.UseFunc(func(w http.ResponseWriter, req *http.Request, next http.HandlerFunc) {
		log.Debugf("[%s] middleware: request dumper", reqid(req))
		defer next(w, req)

		for match, destdir := range self.DebugDumpRequests {
			var filename string

			if fileutil.DirExists(destdir) {
				filename = filepath.Join(destdir, `diecast-req-`+reqid(req)+`.log`)
			} else if fileutil.FileExists(destdir) {
				filename = destdir
			} else {
				return
			}

			if ok, err := filepath.Match(match, req.URL.Path); err == nil && ok || match == `*` {
				if dump, err := os.Create(filename); err == nil {
					dump.Write([]byte(formatRequest(req)))
					dump.Close()
					log.Debugf("wrote request to %v", dump.Name())
				} else {
					log.Warningf("failed to dump request: %v", err)
				}
			}
		}
	})

	// inject global headers
	self.handler.UseFunc(func(w http.ResponseWriter, req *http.Request, next http.HandlerFunc) {
		defer next(w, req)
		log.Debugf("[%s] middleware: inject global headers", reqid(req))

		for k, v := range self.GlobalHeaders {
			if typeutil.IsArray(v) {
				for _, i := range sliceutil.Stringify(v) {
					w.Header().Add(k, i)
				}
			} else {
				w.Header().Set(k, typeutil.String(v))
			}
		}

	})

	// process authenticators
	self.handler.UseFunc(func(w http.ResponseWriter, req *http.Request, next http.HandlerFunc) {
		log.Debugf("[%s] middleware: process authenticators", reqid(req))

		if auth, err := self.Authenticators.Authenticator(req); err == nil {
			if auth != nil {
				if auth.IsCallback(req.URL) {
					auth.Callback(w, req)
					return
				} else if !auth.Authenticate(w, req) {
					return
				}
			}
		} else {
			self.respondError(w, req, err, http.StatusInternalServerError)
		}

		// fallback to proceeding down the middleware chain
		next(w, req)
	})

	// setup CSRF protection (if enabled)
	self.applyCsrfIntercept()

	// cleanup request tracing info
	self.handler.UseFunc(func(w http.ResponseWriter, req *http.Request, next http.HandlerFunc) {
		log.Debugf("[%s] middleware: cleanup request", reqid(req))

		if tm := getRequestTimer(req); tm != nil {
			log.Debugf("[%s] completed: %v", tm.ID, time.Since(tm.StartedAt).Round(time.Microsecond))
		}

		removeRequestTimer(req)
		next(w, req)
	})

	// add favicon.ico handler (if specified)
	faviconRoute := `/` + filepath.Join(self.rp(), `favicon.ico`)

	self.router.HandleFunc(faviconRoute, func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodGet:
			defer req.Body.Close()

			recorder := httptest.NewRecorder()
			recorder.Body = bytes.NewBuffer(nil)

			// before we do anything, make sure this file wouldn't be served
			// through our current application
			self.handleRequest(recorder, req)

			if recorder.Code < 400 {
				for k, vs := range recorder.HeaderMap {
					for _, v := range vs {
						w.Header().Add(k, v)
					}
				}

				io.Copy(w, recorder.Body)
			} else {
				// no favicon cached, so we gotta decode it
				if len(self.faviconImageIco) == 0 {
					var icon io.ReadCloser

					if self.FaviconPath != `` {
						if file, err := self.fs.Open(self.FaviconPath); err == nil {
							icon = file
						}
					}

					if icon == nil {
						w.Header().Set(`Content-Type`, `image/x-icon`)
						w.Write(DefaultFavicon())
						return
					}

					if img, _, err := image.Decode(icon); err == nil {
						buf := bytes.NewBuffer(nil)

						if err := ico.Encode(buf, img); err == nil {
							self.faviconImageIco = buf.Bytes()
						} else {
							log.Debugf("favicon encode: %v", err)
						}
					} else {
						log.Debugf("favicon decode: %v", err)
					}
				}

				if len(self.faviconImageIco) > 0 {
					w.Header().Set(`Content-Type`, `image/x-icon`)
					w.Write(self.faviconImageIco)
				}
			}
		}
	})

	// add action handlers
	for i, action := range self.Actions {
		hndPath := filepath.Join(self.rp(), action.Path)

		if executil.IsRoot() && !executil.EnvBool(`DIECAST_ALLOW_ROOT_ACTIONS`) {
			return fmt.Errorf("Refusing to start as root with actions specified.  Override with the environment variable DIECAST_ALLOW_ROOT_ACTIONS=true")
		}

		if action.Path == `` {
			return fmt.Errorf("Action %d: Must specify a 'path'", i)
		}

		self.router.HandleFunc(hndPath, func(w http.ResponseWriter, req *http.Request) {
			if handler := self.actionForRequest(req); handler != nil {
				handler(w, req)
			} else {
				http.Error(w, fmt.Sprintf("cannot find handler for action"), http.StatusInternalServerError)
			}
		})

		log.Debugf("[actions] Registered %s", hndPath)
	}

	// if we're appending additional trusted certs (for Bindings and other internal HTTP clients)
	if len(self.TrustedRootPEMs) > 0 {
		// get the existing system CA bundle
		if syspool, err := x509.SystemCertPool(); err == nil {
			// append each cert
			for _, pemfile := range self.TrustedRootPEMs {
				// must be a readable PEM file
				if pem, err := fileutil.ReadAll(pemfile); err == nil {
					if !syspool.AppendCertsFromPEM(pem) {
						return fmt.Errorf("Failed to append certificate %s", pemfile)
					}
				} else {
					return fmt.Errorf("Failed to read certificate %s: %v", pemfile, err)
				}
			}

			// this is what http.Client.Transport.TLSClientConfig.RootCAs will become
			self.altRootCaPool = syspool
		} else {
			return fmt.Errorf("Failed to retrieve system CA pool: %v", err)
		}
	}

	return nil
}
