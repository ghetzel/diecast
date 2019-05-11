package diecast

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/typeutil"
)

var reqTimes sync.Map
var timerDescriptions sync.Map

type requestTimer struct {
	ID      string
	Request *http.Request
	Times   map[string]time.Duration
}

func startRequestTimer(req *http.Request) {
	if id := reqid(req); id != `` {
		reqTimes.Store(id, &requestTimer{
			ID:      id,
			Request: req,
			Times:   make(map[string]time.Duration),
		})
	}
}

func describeTimer(key string, desc string) {
	timerDescriptions.Store(key, desc)
}

func reqtime(req *http.Request, key string, took time.Duration) {
	if id := reqid(req); id != `` {
		if v, ok := reqTimes.Load(id); ok {
			if timer, ok := v.(*requestTimer); ok {
				// log.Debugf("[%v] %v=%v", id, key, took)
				timer.Times[key] = took
			}
		}
	}
}

func writeRequestTimerHeaders(server *Server, w http.ResponseWriter, req *http.Request) {
	if server.DisableTimings {
		return
	}

	timings := make([]string, 0)

	if id := reqid(req); id != `` {
		if v, ok := reqTimes.Load(id); ok {
			if timer, ok := v.(*requestTimer); ok {
				for tk, dur := range timer.Times {
					var timing string

					outkey := stringutil.Hyphenate(tk)
					outdur := float64(dur/time.Microsecond) / 1000.0

					if desc, ok := timerDescriptions.Load(tk); ok {
						timing = fmt.Sprintf("%s;desc=%q;dur=%.2f", outkey, typeutil.String(desc), outdur)
					} else {
						timing = fmt.Sprintf("%s;dur=%.2f", outkey, outdur)
					}

					timings = append(timings, timing)
				}
			}
		}

		if len(timings) > 0 {
			w.Header().Set(`Server-Timing`, strings.Join(timings, `, `))
		}
	}
}

func removeRequestTimer(req *http.Request) {
	reqTimes.Delete(reqid(req))
}
