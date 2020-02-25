package diecast

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ghetzel/go-stockutil/httputil"
	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/typeutil"
)

var DefaultProxyMountTimeout = time.Duration(10) * time.Second
var MaxBufferedBodySize int64 = 16535

type ProxyMount struct {
	MountPoint              string                 `json:"-"`
	URL                     string                 `json:"-"`
	Method                  string                 `json:"method,omitempty"`
	Headers                 map[string]interface{} `json:"headers,omitempty"`
	ResponseHeaders         map[string]interface{} `json:"response_headers,omitempty"`
	ResponseCode            int                    `json:"response_code"`
	RedirectOnSuccess       string                 `json:"redirect_on_success"`
	Params                  map[string]interface{} `json:"params,omitempty"`
	Timeout                 interface{}            `json:"timeout,omitempty"`
	PassthroughRequests     bool                   `json:"passthrough_requests"`
	PassthroughHeaders      bool                   `json:"passthrough_headers"`
	PassthroughQueryStrings bool                   `json:"passthrough_query_strings"`
	PassthroughBody         bool                   `json:"passthrough_body"`
	PassthroughErrors       bool                   `json:"passthrough_errors"`
	PassthroughRedirects    bool                   `json:"passthrough_redirects"`
	PassthroughUserAgent    bool                   `json:"passthrough_user_agent"`
	StripPathPrefix         string                 `json:"strip_path_prefix"`
	AppendPathPrefix        string                 `json:"append_path_prefix"`
	Insecure                bool                   `json:"insecure"`
	BodyBufferSize          int64                  `json:"body_buffer_size"`
	CloseConnection         *bool                  `json:"close_connection"`
	Client                  *http.Client
	urlRewriteFrom          string
	urlRewriteTo            string
}

func (self *ProxyMount) GetMountPoint() string {
	return self.MountPoint
}

func (self *ProxyMount) GetTarget() string {
	return self.URL
}

func (self *ProxyMount) WillRespondTo(name string, req *http.Request, requestBody io.Reader) bool {
	return strings.HasPrefix(name, self.GetMountPoint())
}

func (self *ProxyMount) OpenWithType(name string, req *http.Request, requestBody io.Reader) (*MountResponse, error) {
	var id = reqid(req)

	var proxyURI string
	var timeout time.Duration

	if self.Client == nil {
		if t, ok := self.Timeout.(string); ok {
			if tm, err := time.ParseDuration(t); err == nil {
				timeout = tm
			} else {
				log.Warningf("[%s] proxy: INVALID TIMEOUT %q (%v), using default", id, t, err)
				timeout = DefaultProxyMountTimeout
			}
		} else if tm, ok := self.Timeout.(time.Duration); ok {
			timeout = tm
		}

		if timeout == 0 {
			timeout = DefaultProxyMountTimeout
		}

		self.Client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: self.Insecure,
				},
			},
			Timeout: timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if self.PassthroughRedirects {
					return http.ErrUseLastResponse
				} else if self.urlRewriteTo == `` {
					if len(via) > 0 {
						self.urlRewriteFrom = via[len(via)-1].URL.String()
						self.urlRewriteFrom = strings.TrimSuffix(self.urlRewriteFrom, `/`)
						self.urlRewriteTo = req.URL.String()
					}
				}

				return nil
			},
		}
	}

	if req != nil && (self.PassthroughRequests || self.PassthroughQueryStrings) {
		if newURL, err := url.Parse(self.url()); err == nil {
			req.URL.Scheme = newURL.Scheme
			req.URL.Host = newURL.Host

			if newURL.User != nil {
				req.URL.User = newURL.User
			}

			if newURL.Fragment != `` {
				req.URL.Fragment = newURL.Fragment
			}

			// merge incoming query strings with proxy query strings
			var qs = req.URL.Query()

			for newQs, newVs := range newURL.Query() {
				for _, v := range newVs {
					qs.Add(newQs, v)
				}
			}

			req.URL.RawQuery = qs.Encode()

			proxyURI = req.URL.String()
		} else {
			return nil, fmt.Errorf("Failed to parse proxy URL: %v", err)
		}
	} else {
		proxyURI = strings.Join([]string{
			strings.TrimSuffix(self.url(), `/`),
			strings.TrimPrefix(name, `/`),
		}, `/`)
	}

	var method = strings.ToUpper(self.Method)

	if method == `` {
		if req != nil {
			method = req.Method
		} else {
			method = `GET`
		}
	}

	if newReq, err := http.NewRequest(method, proxyURI, nil); err == nil {
		if pp := self.StripPathPrefix; pp != `` {
			newReq.URL.Path = strings.TrimPrefix(newReq.URL.Path, pp)
		}

		if pp := self.AppendPathPrefix; pp != `` {
			newReq.URL.Path = pp + newReq.URL.Path
		}

		if req != nil && (self.PassthroughRequests || self.PassthroughHeaders) {
			for name, values := range req.Header {
				for _, value := range values {
					newReq.Header.Set(name, value)
				}
			}

			if req.Host != `` {
				newReq.Header.Set(`Host`, req.Host)
				newReq.Host = req.Host
			}
		}

		// user-agent is either ours, overridden in explicit headers below, or passthrough from request
		if !self.PassthroughUserAgent {
			newReq.Header.Set(`User-Agent`, DiecastUserAgentString)
		}

		// option to control whether we tell the remote server to close the connection
		if self.CloseConnection != nil {
			if c := *self.CloseConnection; c {
				newReq.Header.Set(`Connection`, `close`)
			} else {
				newReq.Header.Set(`Connection`, `keep-alive`)
			}
		}

		// add explicit headers to new request
		for name, value := range self.Headers {
			newReq.Header.Set(name, typeutil.String(value))
		}

		newReq.Header.Set(`Accept-Encoding`, `identity`)

		// inject params into new request
		for name, value := range self.Params {
			if newReq.URL.Query().Get(name) == `` {
				log.Debugf("[%s] proxy: [Q] %v=%v", id, name, value)
				httputil.SetQ(newReq.URL, name, value)
			}
		}

		if requestBody != nil && (self.PassthroughRequests || self.PassthroughBody) {
			var buf bytes.Buffer
			var bufsz int64 = MaxBufferedBodySize

			if self.BodyBufferSize > 0 {
				bufsz = self.BodyBufferSize
			}

			if n, err := io.CopyN(&buf, requestBody, bufsz); err == nil {
				log.Debugf("[%s] proxy: using streaming request body (body exceeds %d bytes)", id, n)

				// make the upstream request body the aggregate of the already-read portion of the body
				// and the unread remainder of the incoming request body
				newReq.Body = MultiReadCloser(&buf, requestBody)

			} else if err == io.EOF {
				log.Debugf("[%s] proxy: fixed-length request body (%d bytes)", id, buf.Len())
				newReq.Body = MultiReadCloser(&buf)
				newReq.ContentLength = int64(buf.Len())
				newReq.TransferEncoding = []string{`identity`}
			} else {
				return nil, err
			}
		}

		var from = req.Method + ` ` + req.URL.String()
		var to = newReq.Method + ` ` + newReq.URL.String()

		if from == to {
			log.Debugf("[%s] proxy: %s", id, from)
		} else {
			log.Debugf("[%s] proxy: %s", id, from)
			log.Debugf("[%s] proxy: %s (rewritten)", id, to)
		}

		log.Debugf("[%s] proxy: \u256d%s request headers", id, strings.Repeat("\u2500", 56))

		for hdr := range maputil.M(newReq.Header).Iter(maputil.IterOptions{
			SortKeys: true,
		}) {
			log.Debugf("[%s] proxy: \u2502 ${red}%v${reset}: %v", id, hdr.K, stringutil.Elide(strings.Join(hdr.V.Strings(), ` `), 72, `…`))
		}

		log.Debugf("[%s] proxy: \u2570%s end request headers", id, strings.Repeat("\u2500", 56))

		// perform the request
		// -----------------------------------------------------------------------------------------
		log.Debugf("[%s] proxy: sending request to %s://%s", id, newReq.URL.Scheme, newReq.URL.Host)
		var reqStartAt = time.Now()
		response, err := self.Client.Do(newReq)
		log.Debugf("[%s] proxy: responded in %v", id, time.Since(reqStartAt))

		if err == nil {
			if response.Body != nil {
				defer response.Body.Close()
			}

			// add explicit response headers to response
			for name, value := range self.ResponseHeaders {
				response.Header.Set(name, typeutil.String(value))
			}

			// override the response status code (if specified)
			if self.ResponseCode > 0 {
				response.StatusCode = self.ResponseCode
			}

			// provide a header redirect if so requested
			if response.StatusCode < 400 && self.RedirectOnSuccess != `` {
				if response.StatusCode < 300 {
					response.StatusCode = http.StatusTemporaryRedirect
				}

				response.Header.Set(`Location`, self.RedirectOnSuccess)
			}

			log.Debugf("[%s] proxy: HTTP %v", id, response.Status)
			log.Debugf("[%s] proxy: \u256d%s response headers", id, strings.Repeat("\u2500", 56))

			for hdr := range maputil.M(response.Header).Iter(maputil.IterOptions{
				SortKeys: true,
			}) {
				log.Debugf("[%s] proxy: \u2502 ${blue}%v${reset}: %v", id, hdr.K, stringutil.Elide(strings.Join(hdr.V.Strings(), ` `), 72, `…`))
			}

			log.Debugf("[%s] proxy: \u2570%s end response headers", id, strings.Repeat("\u2500", 56))

			log.Infof(
				"[%s] proxy: %s responded with: %v (Content-Length: %v)",
				id,
				to,
				response.Status,
				response.ContentLength,
			)

			if response.StatusCode < 400 || self.PassthroughErrors {
				var responseBody io.Reader

				if body, err := httputil.DecodeResponse(response); err == nil {
					responseBody = body

					// whatever the encoding was before, it's definitely "identity" now
					response.Header.Set(`Content-Encoding`, `identity`)
				} else {
					return nil, err
				}

				if data, err := ioutil.ReadAll(responseBody); err == nil {
					var payload = bytes.NewReader(data)

					// correct the length, which is now potentially decompressed and longer
					// than the original response claims
					response.Header.Set(`Content-Length`, typeutil.String(payload.Size()))

					var mountResponse = NewMountResponse(name, payload.Size(), payload)
					mountResponse.StatusCode = response.StatusCode
					mountResponse.ContentType = response.Header.Get(`Content-Type`)

					for k, v := range response.Header {
						mountResponse.Metadata[k] = strings.Join(v, `,`)
					}

					return mountResponse, nil
				} else {
					return nil, fmt.Errorf("proxy response: %v", err)
				}
			} else {
				if data, err := ioutil.ReadAll(response.Body); err == nil {
					for _, line := range stringutil.SplitLines(data, "\n") {
						log.Debugf("[%s] proxy: [B] %s", id, line)
					}
				}

				log.Debugf("[%s] proxy: %s %s: %s", id, method, newReq.URL, response.Status)

				return nil, MountHaltErr
			}
		} else {
			return nil, err
		}
	} else {
		return nil, err
	}
}

func (self *ProxyMount) url() string {
	var uri = self.URL

	if from := self.urlRewriteFrom; from != `` {
		if to := self.urlRewriteTo; to != `` {
			uri = strings.Replace(uri, from, to, 1)

			log.Debugf("Rewriting %v to %v due to earlier redirect", self.urlRewriteFrom, self.urlRewriteTo)
		}
	}

	return uri
}

func (self *ProxyMount) String() string {
	return fmt.Sprintf(
		"%v -> %v %v (passthrough requests=%v errors=%v)",
		self.MountPoint,
		strings.ToUpper(sliceutil.OrString(self.Method, `<method>`)),
		self.url(),
		self.PassthroughRequests,
		self.PassthroughErrors,
	)
}

func (self *ProxyMount) Open(name string) (http.File, error) {
	return openAsHttpFile(self, name)
}
