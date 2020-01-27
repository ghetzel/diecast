package diecast

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/ghetzel/go-stockutil/httputil"
	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/typeutil"
)

type RequestAuthenticator struct {
	config     *AuthenticatorConfig
	remotes    []string
	methods    []string
	headers    map[string]string
	remoteNets map[string]*net.IPNet
}

func NewRequestAuthenticator(config *AuthenticatorConfig) (*RequestAuthenticator, error) {
	auth := &RequestAuthenticator{
		config: config,
		methods: sliceutil.MapString(config.O(`methods`).Strings(), func(i int, value string) string {
			return strings.ToUpper(value)
		}),
		remotes:    config.O(`remotes`).Strings(),
		headers:    make(map[string]string),
		remoteNets: make(map[string]*net.IPNet),
	}

	for hdr, pattern := range config.O(`headers`).MapNative() {
		auth.headers[hdr] = typeutil.String(pattern)
	}

	for _, remote := range auth.remotes {
		if strings.Contains(remote, `/`) {
			if _, ipnet, err := net.ParseCIDR(remote); err == nil {
				auth.remoteNets[remote] = ipnet
			} else {
				return nil, fmt.Errorf("bad remote %q: %v", remote, err)
			}
		}
	}

	return auth, nil
}

func (self *RequestAuthenticator) Name() string {
	if self.config != nil && self.config.Name != `` {
		return self.config.Name
	} else {
		return `RequestAuthenticator`
	}
}

func (self *RequestAuthenticator) IsCallback(_ *url.URL) bool {
	return false
}

func (self *RequestAuthenticator) Callback(w http.ResponseWriter, req *http.Request) {

}

func (self *RequestAuthenticator) Authenticate(w http.ResponseWriter, req *http.Request) bool {
	if len(self.methods) > 0 {
		if !sliceutil.ContainsString(self.methods, req.Method) {
			httputil.RequestSetValue(req, ContextErrorKey, fmt.Sprintf("HTTP method %s is not permitted", req.Method))
			return false
		}
	}

	// if remotes are specified, one must match
	if len(self.remotes) > 0 {
		if addr, _ := stringutil.SplitPair(req.RemoteAddr, `:`); addr != `` {
			for i, remote := range self.remotes {
				if addr == remote {
					log.Debugf(
						"[%s] request-auth: permitting address %v (in remote %d; exact match: %v)",
						reqid(req),
						addr,
						i,
						remote,
					)
					return true
				} else if ipnet, ok := self.remoteNets[remote]; ok && ipnet != nil {
					if ip := net.ParseIP(addr); ip != nil {
						if ipnet.Contains(ip) {
							log.Debugf(
								"[%s] request-auth: permitting address %v (in remote %d; network: %v)",
								reqid(req),
								addr,
								i,
								ipnet,
							)
							return true
						}
					}
				}
			}

			httputil.RequestSetValue(req, ContextErrorKey, fmt.Sprintf("Address %s is not permitted", addr))
			return false
		} else {
			httputil.RequestSetValue(req, ContextErrorKey, "Address could not be determined")
			return false
		}
	}

	return true
}
