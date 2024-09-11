package internal

import (
	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/typeutil"
	"io/fs"
	"net/http"
	"time"
)

type Contextable interface {
	Code() int
	Critical(args ...interface{})
	Criticalf(format string, args ...interface{})
	Data() map[string]interface{}
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
	Done() time.Duration
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Eval(value interface{}) (typeutil.Variant, error)
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	Get(key string, fallback ...interface{}) typeutil.Variant
	Header() http.Header
	ID() string
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Log(level log.Level, args ...interface{})
	Logf(level log.Level, format string, args ...interface{})
	MarkTemplateSeen(name string) bool
	Notice(args ...interface{})
	Noticef(format string, args ...interface{})
	Open(name string) (fs.File, error)
	Panic(args ...interface{})
	Panicf(format string, args ...interface{})
	Pop(key string) typeutil.Variant
	PushValue(key string, value interface{})
	Request() *http.Request
	RequestBasename() string
	SetTypeHint(hint string)
	SetValue(key string, value interface{})
	StartHTTP(wr http.ResponseWriter, req *http.Request)
	T(value interface{}) typeutil.Variant
	TypeHint() string
	Warning(args ...interface{})
	Warningf(format string, args ...interface{})
	WasTemplateSeen(name string) bool
	Write(b []byte) (int, error)
	WriteHeader(statusCode int)
}
