package diecast

import (
	"bytes"
	"fmt"
	"mime"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ghetzel/go-stockutil/convutil"
	"github.com/ghetzel/go-stockutil/fileutil"
	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/typeutil"
)

type RequestIdentFunc func(*http.Request) string

var DefaultContextTypeHint = `application/octet-stream`
var DefaultContextDir = `.`
var RequestIdentifierFunc RequestIdentFunc

// A Context represents everything necessary to handle the request for a single resource, including
// validating the request may proceed, locating and retrieving the data, and performing any
// post-processing of that data before it is returned to the requestor.
type Context struct {
	*maputil.Map
	wr           http.ResponseWriter
	req          *http.Request
	fs           http.FileSystem
	startedAt    time.Time
	statusCode   int
	bytesWritten int64
	slock        sync.Mutex
	mimeHint     string
	id           string
}

func NewContext(fs http.FileSystem) *Context {
	var ctx = &Context{
		fs: fs,
	}

	return ctx.reset()
}

// Initialize all internal state such that a new request can begin via Start().
func (self *Context) reset() *Context {
	if self.fs == nil {
		self.fs = http.Dir(DefaultContextDir)
	}

	self.id = ``
	self.id = self.ID()
	self.Map = maputil.M(sync.Map{})
	self.wr = nil
	self.req = nil
	self.startedAt = time.Time{}
	self.bytesWritten = 0
	self.mimeHint = DefaultContextTypeHint
	self.statusCode = http.StatusOK

	return self
}

// Anything to do to the response immediately before rendering begins, after which point we lose control of the response.
func (self *Context) finalizeBeforeRender() {
	self.Header().Set(`X-Diecast-Request`, self.ID())
}

// Retrieve the media type of the MIME type the response should have.
func (self *Context) TypeHint() string {
	if mt, _, err := mime.ParseMediaType(self.mimeHint); err == nil && mt != `` {
		return mt
	} else {
		return typeutil.OrString(self.mimeHint, DefaultContextTypeHint)
	}
}

// Override the MIME type that describes the response for the current context.
func (self *Context) SetTypeHint(hint string) {
	self.mimeHint = hint

	if self.wr != nil {
		self.wr.Header().Set(`Content-Type`, self.mimeHint)
	}
}

// Start tracking a specific request+response pair.  Mark the request as completed with Done().
func (self *Context) Start(wr http.ResponseWriter, req *http.Request) *Context {
	self.slock.Lock()
	defer self.slock.Unlock()

	self.wr = wr
	self.req = req
	self.startedAt = time.Now()

	self.SetTypeHint(fileutil.GetMimeType(self.req.URL.Path, self.mimeHint))
	self.Debugf("start request: %s %v", self.req.Method, self.req.URL)

	for kv := range maputil.M(req.Header).Iter(maputil.IterOptions{
		SortKeys: true,
	}) {
		self.Debugf("  % -32s: %v", kv.K, kv.Value)
	}

	return self
}

// Mark the request as completed.  If desired, the context instance can be reused with a subsequent
// call to Start().
func (self *Context) Done() time.Duration {
	self.slock.Lock()
	defer func() {
		self.reset()
		self.slock.Unlock()
	}()

	var rhdr = self.wr.Header()
	var took = time.Since(self.startedAt)
	var code = self.Code()
	var lvl log.Level

	if code < 400 {
		lvl = log.NOTICE
	} else {
		lvl = log.WARNING
	}

	self.Logf(
		lvl,
		"responded HTTP %d %s (%v @ %v)",
		code,
		http.StatusText(code),
		convutil.Bytes(self.bytesWritten),
		took.Round(time.Microsecond),
	)

	for kv := range maputil.M(rhdr).Iter(maputil.IterOptions{
		SortKeys: true,
	}) {
		self.Debugf("  % -32s: %v", kv.K, kv.Value)
	}

	return took
}

// Return the unique request ID.
func (self *Context) ID() string {
	if self.id != `` {
		return self.id
	}

	// honor package-level custom identifier function
	if rifn := RequestIdentifierFunc; rifn != nil {
		if id := rifn(self.req); id != `` {
			return id
		}
	}

	// attempt to reuse existing tracing IDs seen in the wild
	if req := self.req; req != nil {
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
		}
	}

	return stringutil.UUID().Base58()
}

// Open a file in the underlying http.FileSystem.
func (self *Context) Open(name string) (http.File, error) {
	if self.fs == nil {
		panic("no filesystem associated with context")
	}

	self.Debugf("fs: open %q", name)
	return self.fs.Open(name)
}

// Return the http.Request associated with this context.  This function will panic if Start() was
// not previously called with a non-nil http.Request.
func (self *Context) Request() *http.Request {
	if self.req == nil {
		panic("no request associated with context")
	}

	return self.req
}

// Return the header set from the underlying http.ResponseWriter.
func (self *Context) Header() http.Header {
	return self.wr.Header()
}

// Passthrough a Write to the underlying http.ResponseWriter.
func (self *Context) Write(b []byte) (int, error) {
	var n, err = self.wr.Write(b)
	self.bytesWritten += int64(n)
	return n, err
}

// Write the response status code and keep a copy for later inspection.
func (self *Context) WriteHeader(statusCode int) {
	self.statusCode = statusCode
	self.wr.WriteHeader(self.Code())
}

// Return a usable HTTP status code for the reponse.
func (self *Context) Code() int {
	return typeutil.OrNInt(self.statusCode, http.StatusOK)
}

// Evaluates the given value as a template if it is one, and returns the resulting value.  If the input
// is not a string that contains template tags, the value will be enclosed unmodified in the returned
// typeutil.Variant, accessible via its Value field.
func (self *Context) Eval(value interface{}) (typeutil.Variant, error) {
	if value == nil {
		return typeutil.Nil(), nil
	} else if typeutil.IsKindOfString(value) {
		if ts := typeutil.String(value); strings.Contains(ts, Delimiters[0]) && strings.Contains(ts, Delimiters[1]) {
			if tmpl, err := ParseTemplateString(ts); err == nil {
				var buf bytes.Buffer

				if err := tmpl.Render(self, &buf); err == nil {
					return typeutil.V(buf.Bytes()), nil
				} else {
					return typeutil.Nil(), err
				}
			} else {
				return typeutil.Nil(), err
			}
		}
	}

	return typeutil.V(value), nil
}

// A simple inline context-aware template string evaluator.
func (self *Context) T(value interface{}) typeutil.Variant {
	if v, err := self.Eval(value); err == nil {
		return v
	} else {
		return typeutil.Nil()
	}
}

// The remaining functions implement the logging pseudointerface in go-stockutil/log such that
// all context-specific log statements can be intercepted, formatted, and processed.

func (self *Context) Log(level log.Level, args ...interface{}) {
	log.Log(level, append([]interface{}{
		fmt.Sprintf("%s: ", self.ID()),
	}, args...)...)
}

func (self *Context) Logf(level log.Level, format string, args ...interface{}) {
	log.Logf(level, "%s: "+format, append([]interface{}{self.ID()}, args...)...)
}

func (self *Context) Debug(args ...interface{}) {
	self.Log(log.DEBUG, args...)
}

func (self *Context) Info(args ...interface{}) {
	self.Log(log.INFO, args...)
}

func (self *Context) Notice(args ...interface{}) {
	self.Log(log.NOTICE, args...)
}

func (self *Context) Warning(args ...interface{}) {
	self.Log(log.WARNING, args...)
}

func (self *Context) Error(args ...interface{}) {
	self.Log(log.ERROR, args...)
}

func (self *Context) Fatal(args ...interface{}) {
	self.Log(log.FATAL, args...)
}

func (self *Context) Critical(args ...interface{}) {
	self.Log(log.CRITICAL, args...)
}

func (self *Context) Panic(args ...interface{}) {
	self.Log(log.PANIC, args...)
}

func (self *Context) Debugf(format string, args ...interface{}) {
	self.Logf(log.DEBUG, format, args...)
}

func (self *Context) Infof(format string, args ...interface{}) {
	self.Logf(log.INFO, format, args...)
}

func (self *Context) Noticef(format string, args ...interface{}) {
	self.Logf(log.NOTICE, format, args...)
}

func (self *Context) Warningf(format string, args ...interface{}) {
	self.Logf(log.WARNING, format, args...)
}

func (self *Context) Errorf(format string, args ...interface{}) {
	self.Logf(log.ERROR, format, args...)
}

func (self *Context) Fatalf(format string, args ...interface{}) {
	self.Logf(log.FATAL, format, args...)
}

func (self *Context) Criticalf(format string, args ...interface{}) {
	self.Logf(log.CRITICAL, format, args...)
}

func (self *Context) Panicf(format string, args ...interface{}) {
	self.Logf(log.PANIC, format, args...)
}
