package internal

import (
	"io/fs"
	"net/http"
	"time"

	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/typeutil"
)

type Contextable interface {
	Code() int
	Critical(args ...any)
	Criticalf(format string, args ...any)
	Data() map[string]any
	Debug(args ...any)
	Debugf(format string, args ...any)
	Done() time.Duration
	Error(args ...any)
	Errorf(format string, args ...any)
	Eval(value any) (typeutil.Variant, error)
	Fatal(args ...any)
	Fatalf(format string, args ...any)
	Get(key string, fallback ...any) typeutil.Variant
	Header() http.Header
	ID() string
	Info(args ...any)
	Infof(format string, args ...any)
	Log(level log.Level, args ...any)
	Logf(level log.Level, format string, args ...any)
	MarkTemplateSeen(name string) bool
	Notice(args ...any)
	Noticef(format string, args ...any)
	Open(name string) (fs.File, error)
	Panic(args ...any)
	Panicf(format string, args ...any)
	Pop(key string) typeutil.Variant
	PushValue(key string, value any)
	Request() *http.Request
	RequestBasename() string
	SetTypeHint(hint string)
	SetValue(key string, value any)
	StartHTTP(wr http.ResponseWriter, req *http.Request)
	T(value any) typeutil.Variant
	TypeHint() string
	Warning(args ...any)
	Warningf(format string, args ...any)
	WasTemplateSeen(name string) bool
	Write(b []byte) (int, error)
	WriteHeader(statusCode int)
}
