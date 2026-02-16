package internal

import (
	"io/fs"
	"net/http"
	"time"

	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/maputil"
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
	Increment(key string, value float64) float64
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

// Evaluate a value against the given Contextable or panic.
func MustEval(ctx Contextable, tpl any) any {
	if v, err := ctx.Eval(tpl); err == nil {
		return v.Value
	} else {
		panic(err.Error())
	}
}

func MapEval(ctx Contextable, data map[string]any) map[string]any {
	log.DumpJSON(data)
	return maputil.Apply(data, func(key []string, value any) (any, bool) {
		if !typeutil.IsEmpty(value) {
			if v, err := ctx.Eval(value); err == nil {
				value = v.Value
			} else {
				ctx.Warningf("invalid template %q: %v", value, err)
				value = nil
			}
		}

		return value, true
	})
}

func StringMapEval(ctx Contextable, data map[string]any) map[string]string {
	return maputil.Stringify(MapEval(ctx, data))
}
