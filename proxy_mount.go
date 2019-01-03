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

	"github.com/ghetzel/diecast/util"
	"github.com/ghetzel/go-stockutil/httputil"
	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/typeutil"
)

var DefaultProxyMountTimeout = time.Duration(10) * time.Second
var MaxBufferedBodySize = 16535

type ProxyMount struct {
	MountPoint          string                 `json:"-"`
	URL                 string                 `json:"-"`
	Method              string                 `json:"method,omitempty"`
	Headers             map[string]interface{} `json:"headers,omitempty"`
	Params              map[string]interface{} `json:"params,omitempty"`
	Timeout             time.Duration          `json:"timeout,omitempty"`
	PassthroughRequests bool                   `json:"passthrough_requests"`
	PassthroughErrors   bool                   `json:"passthrough_errors"`
	StripPathPrefix     string                 `json:"strip_path_prefix"`
	Insecure            bool                   `json:"insecure"`
	Client              *http.Client
	urlRewriteFrom      string
	urlRewriteTo        string
}

func (self *ProxyMount) GetMountPoint() string {
	return self.MountPoint
}

func (self *ProxyMount) WillRespondTo(name string, req *http.Request, requestBody io.Reader) bool {
	return strings.HasPrefix(name, self.GetMountPoint())
}

func (self *ProxyMount) OpenWithType(name string, req *http.Request, requestBody io.Reader) (*MountResponse, error) {
	var proxyURI string

	if self.Client == nil {
		if self.Timeout == 0 {
			self.Timeout = DefaultProxyMountTimeout
		}

		self.Client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: self.Insecure,
				},
			},
			Timeout: self.Timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if self.urlRewriteTo == `` {
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

	if self.Method == `` {
		self.Method = `get`
	}

	if req != nil && self.PassthroughRequests {
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
			qs := req.URL.Query()

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

	method := strings.ToUpper(self.Method)

	if req != nil && self.PassthroughRequests {
		method = req.Method
	}

	if newReq, err := http.NewRequest(method, proxyURI, nil); err == nil {
		if pp := self.StripPathPrefix; pp != `` {
			newReq.URL.Path = strings.TrimPrefix(newReq.URL.Path, pp)
		}

		if req != nil && self.PassthroughRequests {
			for name, values := range req.Header {
				for _, value := range values {
					newReq.Header.Set(name, value)
				}
			}
		}

		// add explicit headers to new request
		for name, value := range self.Headers {
			newReq.Header.Set(name, typeutil.String(value))
		}

		// inject params into new request
		for name, value := range self.Params {
			if newReq.URL.Query().Get(name) == `` {
				log.Debugf("  [Q] %v=%v", name, value)
				httputil.SetQ(newReq.URL, name, value)
			}
		}

		log.Debugf("  Handled by %v", self)

		if requestBody != nil && self.PassthroughRequests {
			var buf bytes.Buffer

			if n, err := io.CopyN(&buf, requestBody, int64(MaxBufferedBodySize)); err == nil {
				log.Debugf("  using streaming request body (body exceeds %d bytes)", n)

				// make the upstream request body the aggregate of the already-read portion of the body
				// and the unread remainder of the incoming request body
				newReq.Body = util.NewChainableReader(&buf, requestBody)

			} else if err == io.EOF {
				log.Debugf("  fixed-length request body (%d bytes)", buf.Len())
				newReq.Body = ioutil.NopCloser(&buf)
				newReq.ContentLength = int64(buf.Len())
				newReq.TransferEncoding = []string{`identity`}
			} else {
				return nil, err
			}
		}

		log.Infof("  proxying '%v %v' to '%v %v'", req.Method, req.URL, newReq.Method, newReq.URL)
		log.Debugf("  %v %v", newReq.Method, newReq.URL)

		for k, v := range newReq.Header {
			log.Debugf("  [H] %v=%v", k, strings.Join(v, ` `))
		}

		if response, err := self.Client.Do(newReq); err == nil {
			log.Debugf("  [R] %v", response.Status)

			for k, v := range response.Header {
				log.Debugf("  [R]   %v: %v", k, strings.Join(v, ` `))
			}

			log.Infof(
				"%v %v responded with: %v (Content-Length: %v)",
				newReq.Method,
				newReq.URL,
				response.Status,
				response.ContentLength,
			)

			if response.StatusCode < 400 || self.PassthroughErrors {
				var responseBody io.Reader
				defer response.Body.Close()

				if body, err := httputil.DecodeResponse(response); err == nil {
					responseBody = body
					response.Header.Set(`Content-Encoding`, `identity`)
				} else {
					return nil, err
				}

				if data, err := ioutil.ReadAll(responseBody); err == nil {
					payload := bytes.NewReader(data)
					mountResponse := NewMountResponse(name, payload.Size(), payload)
					mountResponse.StatusCode = response.StatusCode
					mountResponse.ContentType = response.Header.Get(`Content-Type`)

					for k, v := range response.Header {
						mountResponse.Metadata[k] = strings.Join(v, `,`)
					}

					return mountResponse, nil
				} else {
					return nil, err
				}
			} else {
				// log.Debugf("  %s %s: %s", method, newReq.URL, response.Status)
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
	uri := self.URL

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
		strings.ToUpper(sliceutil.OrString(self.Method, `get`)),
		self.url(),
		self.PassthroughRequests,
		self.PassthroughErrors,
	)
}

func (self *ProxyMount) Open(name string) (http.File, error) {
	return openAsHttpFile(self, name)
}
