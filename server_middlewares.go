package diecast

import (
	"bytes"
	"crypto/x509"
	"fmt"
	"image"
	"io"
	"io/ioutil"
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
	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/typeutil"
	base58 "github.com/jbenet/go-base58"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

// A function that receives the current request, ResponseWriter, and returns whether to call the next middleware
// in the stack (true) or to stop processing the request immediately (false).
type Middleware func(w http.ResponseWriter, req *http.Request) bool

func (self *Server) setupServer() error {
	if err := self.configureTls(); err != nil {
		return err
	}

	if err := self.registerInternalRoutes(); err != nil {
		return err
	}

	self.BeforeHandlers = []Middleware{
		self.middlewareStartRequest,
		self.middlewareParseRequestBody,
		self.middlewareDebugRequest,
		self.middlewareInjectHeaders,
		self.middlewareProcessAuthenticators,
		self.middlewareCsrf,
	}

	self.AfterHandlers = []http.HandlerFunc{
		self.afterFinalizeAndLog,
	}

	return nil
}

func (self *Server) traceName(candidate string) string {
	if jc := self.JaegerConfig; jc != nil && jc.Enable {
		for _, mapping := range jc.OperationsMappings {
			if newName, matched := mapping.TraceName(candidate); matched {
				return newName
			}
		}
	}

	return candidate
}

func (self *Server) traceNameFromRequest(req *http.Request) string {
	return self.traceName(fmt.Sprintf("%s %s", req.Method, req.URL.Path))
}

func (self *Server) requestIdFromRequest(req *http.Request) string {
	if id := req.Header.Get(`traceparent`); id != `` {
		return id
	} else if id := maputil.M(
		maputil.Split(req.Header.Get(`x-amzn-trace-id`), `=`, `;`),
	).String(`Root`); id != `` {
		return id
	} else if id := req.Header.Get(`uber-trace-id`); id != `` {
		return id
	} else if id := req.Header.Get(`apigw-requestid`); id != `` {
		return id
	} else {
		return base58.Encode(stringutil.UUID().Bytes())
	}
}

// setup request (generate ID, intercept ResponseWriter to get status code, set context variables)
func (self *Server) middlewareStartRequest(w http.ResponseWriter, req *http.Request) bool {
	var requestId = self.requestIdFromRequest(req)

	log.Debugf("[%s] %s", requestId, strings.Repeat(`-`, 69))
	log.Debugf("[%s] %s %s (%s)", requestId, req.Method, req.RequestURI, req.RemoteAddr)

	// setup opentracing for this request (if we should)
	if self.opentrace != nil {
		if traceName := self.traceNameFromRequest(req); traceName != `` {
			var span opentracing.Span

			// continue an existing span or start a new one
			if wctx, err := self.opentrace.Extract(
				opentracing.HTTPHeaders,
				opentracing.HTTPHeadersCarrier(req.Header),
			); err == nil {
				span = self.opentrace.StartSpan(traceName, ext.RPCServerOption(wctx))
			} else {
				span = self.opentrace.StartSpan(traceName)
			}

			if span != nil {
				span.SetBaggageItem(`diecast.request_id`, requestId)
				httputil.RequestSetValue(req, JaegerSpanKey, span)
			}

			log.Debugf("[%s] middleware: trace operation: %s", requestId, traceName)
		}
	}

	httputil.RequestSetValue(req, ContextRequestKey, requestId)
	w.Header().Set(`X-Diecast-Request-ID`, requestId)

	// setup request tracing info
	startRequestTimer(req)

	if self.stopping {
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		return false
	}

	return true
}

func (self *Server) middlewareParseRequestBody(w http.ResponseWriter, req *http.Request) bool {
	if reqbody(req) != nil {
		return true
	}

	// request body
	// ------------------------------------------------------------------------
	var body = new(RequestBody)

	body.Length = req.ContentLength
	body.Original = req.Body

	httputil.RequestSetValue(req, RequestBodyKey, body)

	if req.Body != nil && req.ContentLength > 0 {
		defer req.Body.Close()

		if data, err := ioutil.ReadAll(req.Body); err == nil {
			body.Raw = data
			body.String = string(data)
			body.Loaded = true

			req.Body = body
			req.ContentLength = body.Length
			req.Header.Del(`Content-Encoding`)
			req.Header.Del(`Transfer-Encoding`)
			req.Header.Set(`Content-Length`, typeutil.String(req.ContentLength))

			if err := httputil.ParseRequest(req, &body.Parsed); err != nil {
				body.Error = err.Error()
			}
		} else if err != io.EOF {
			body.Error = err.Error()
		}
	}

	return true
}

// handle request dumper (for debugging)
func (self *Server) middlewareDebugRequest(w http.ResponseWriter, req *http.Request) bool {
	if len(self.DebugDumpRequests) > 0 {
		log.Debugf("[%s] middleware: request dumper", reqid(req))
		for match, destdir := range self.DebugDumpRequests {
			var filename string

			if fileutil.DirExists(destdir) {
				filename = filepath.Join(destdir, `diecast-req-`+reqid(req)+`.log`)
			} else if fileutil.FileExists(destdir) {
				filename = destdir
			} else {
				break
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
	}

	return true
}

// inject global headers
func (self *Server) middlewareInjectHeaders(w http.ResponseWriter, req *http.Request) bool {
	if len(self.GlobalHeaders) > 0 {
		log.Debugf("[%s] middleware: inject global headers", reqid(req))

		for k, v := range self.GlobalHeaders {
			if typeutil.IsArray(v) {
				for _, i := range sliceutil.Stringify(v) {
					w.Header().Add(k, i)
				}
			} else if typeutil.IsMap(v) {
				w.Header().Set(k, fancyMapJoin(v))
			} else {
				w.Header().Set(k, typeutil.String(v))
			}
		}
	}

	return true
}

// process authenticators
func (self *Server) middlewareProcessAuthenticators(w http.ResponseWriter, req *http.Request) bool {
	if len(self.Authenticators) > 0 {
		log.Debugf("[%s] middleware: process authenticators", reqid(req))

		if auth, err := self.Authenticators.Authenticator(req); err == nil {
			if auth != nil {
				if auth.IsCallback(req.URL) {
					auth.Callback(w, req)
					return false
				} else if !auth.Authenticate(w, req) {
					httputil.RequestSetValue(req, ContextStatusKey, http.StatusForbidden)
					return false
				}
			}
		} else {
			self.respondError(w, req, err, http.StatusInternalServerError)
		}
	}

	// fallback to proceeding down the middleware chain
	return true
}

// cleanup request tracing info
func (self *Server) afterFinalizeAndLog(w http.ResponseWriter, req *http.Request) {
	// log.Debugf("[%s] after: finalize and log request", reqid(req))
	var took time.Duration

	if tm := getRequestTimer(req); tm != nil {
		took = time.Since(tm.StartedAt).Round(time.Microsecond)
		log.Debugf("[%s] completed: %v", tm.ID, took)
		httputil.RequestSetValue(req, `duration`, took)
	}

	// finish up and close out trace
	if ot, ok := httputil.RequestGetValue(req, JaegerSpanKey).Value.(opentracing.Span); ok {
		var interceptor = reqres(req)
		ot.SetTag(`http.status_code`, interceptor.code)
		ot.SetTag(`http.response_content_length`, interceptor.bytesWritten)
		ot.Finish()
	}

	self.logreq(w, req)
	removeRequestTimer(req)
}

// adds a pile of TLS configuration for the benefit of the various HTTP clients uses to do things so
// that you have an alternative to "insecure: true"; a ray of sunlight in the dark sky of practical modern web cryptosystems.
func (self *Server) configureTls() error {
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

// adds routes for things like favicon and actions.
func (self *Server) registerInternalRoutes() error {
	// setup handler for template tree
	self.mux.HandleFunc(self.rp()+`/`, self.handleRequest)

	// add favicon.ico handler (if specified)
	var faviconRoute = `/` + filepath.Join(self.rp(), `favicon.ico`)

	self.mux.HandleFunc(faviconRoute, func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodGet:
			defer req.Body.Close()

			var recorder = httptest.NewRecorder()
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
						return
					}

					if img, _, err := image.Decode(icon); err == nil {
						var buf = bytes.NewBuffer(nil)

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
		var hndPath = filepath.Join(self.rp(), action.Path)

		if executil.IsRoot() && !executil.EnvBool(`DIECAST_ALLOW_ROOT_ACTIONS`) {
			return fmt.Errorf("Refusing to start as root with actions specified.  Override with the environment variable DIECAST_ALLOW_ROOT_ACTIONS=true")
		}

		if action.Path == `` {
			return fmt.Errorf("Action %d: Must specify a 'path'", i)
		}

		self.mux.HandleFunc(hndPath, func(w http.ResponseWriter, req *http.Request) {
			if handler := self.actionForRequest(req); handler != nil {
				handler(w, req)
			} else {
				http.Error(w, fmt.Sprintf("cannot find handler for action"), http.StatusInternalServerError)
			}
		})

		log.Debugf("[actions] Registered %s", hndPath)
	}

	return nil
}
