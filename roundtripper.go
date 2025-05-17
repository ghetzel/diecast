package diecast

import (
	"context"
	"net"
	"net/http"

	"github.com/ghetzel/go-stockutil/httputil"
)

type transportAwareRoundTripper struct {
	transport *http.Transport
}

func (rtt *transportAwareRoundTripper) SetTransport(t *http.Transport) {
	rtt.transport = t
}

func (rtt *transportAwareRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if rtt.transport == nil {
		rtt.transport = http.DefaultTransport.(*http.Transport)
	}

	var perReqTransport = rtt.transport.Clone()

	if socketPath := httputil.RequestGetValue(req, `diecastSocketPath`); socketPath.String() != `` {
		perReqTransport.DialContext = func(_ context.Context, _, _ string) (net.Conn, error) {
			var network string

			if n := httputil.RequestGetValue(req, `diecastNetwork`).String(); n != `` {
				network = n
			} else {
				network = `unix`
			}

			return net.Dial(network, socketPath.String())
		}
	}

	return perReqTransport.RoundTrip(req)
}
