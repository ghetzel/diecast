package diecast

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"

	"github.com/ghetzel/go-stockutil/fileutil"
	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/typeutil"
)

type Context struct {
	*maputil.Map
	wr  http.ResponseWriter
	req *http.Request
	fs  http.FileSystem
}

func NewContext(wr http.ResponseWriter, req *http.Request, fs http.FileSystem) *Context {
	if wr == nil {
		wr = httptest.NewRecorder()
	}

	if req == nil {
		req = httptest.NewRequest(`GET`, `/`, nil)
	}

	if fs == nil {
		fs = http.Dir(`.`)
	}

	return &Context{
		Map: maputil.M(sync.Map{}),
		wr:  wr,
		req: req,
		fs:  fs,
	}
}

func (self *Context) ID() string {
	return `potato`
}

func (self *Context) Open(name string) (http.File, error) {
	return self.fs.Open(name)
}

func (self *Context) Request() *http.Request {
	return self.req
}

func (self *Context) Header() http.Header {
	return self.wr.Header()
}

func (self *Context) Write(b []byte) (int, error) {
	return self.wr.Write(b)
}

func (self *Context) WriteHeader(statusCode int) {
	self.wr.WriteHeader(statusCode)
}

func (self *Context) Eval(value interface{}) (typeutil.Variant, error) {
	if value == nil {
		return typeutil.Nil(), nil
	} else if typeutil.IsKindOfString(value) {
		if ts := typeutil.String(value); strings.Contains(ts, Delimiters[0]) && strings.Contains(ts, Delimiters[1]) {
			if tmpl, unread, err := ParseTemplateString(ts); err == nil {
				var buf bytes.Buffer

				if err := tmpl.Render(self, &buf); err == nil {
					return typeutil.V(buf.Bytes()), nil
				} else {
					return typeutil.Nil(), err
				}
			} else {
				return typeutil.V(fileutil.Cat(unread)), err
			}
		}
	}

	return typeutil.V(value), nil
}

func (self *Context) T(value interface{}) typeutil.Variant {
	if v, err := self.Eval(value); err == nil {
		return v
	} else {
		return typeutil.Nil()
	}
}

func (self *Context) Log(level log.Level, args ...interface{}) {
	log.Log(level, append([]interface{}{
		fmt.Sprintf("▶ % -16s ▶", self.ID()),
	}, args...)...)
}

func (self *Context) Logf(level log.Level, format string, args ...interface{}) {
	log.Logf(level, fmt.Sprintf("▶ % -16s ▶ %s", self.ID(), format), args...)
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
