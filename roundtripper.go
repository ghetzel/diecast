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

func (self *transportAwareRoundTripper) SetTransport(t *http.Transport) {
	self.transport = t
}

func (self *transportAwareRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if self.transport == nil {
		self.transport = http.DefaultTransport.(*http.Transport)
	}

	var perReqTransport = self.transport.Clone()

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
