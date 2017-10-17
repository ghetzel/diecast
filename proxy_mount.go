package diecast

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var DefaultProxyMountTimeout = time.Duration(10) * time.Second

type responseFile struct {
	http.File
	payload io.ReadSeeker
	name    string
	size    int64
}

func (self *responseFile) Read(p []byte) (int, error) {
	if self.payload == nil {
		return 0, fmt.Errorf("Cannot read from closed response")
	} else {
		return self.payload.Read(p)
	}
}

func (self *responseFile) Seek(offset int64, whence int) (int64, error) {
	if self.payload == nil {
		return 0, fmt.Errorf("Cannot seek in closed response")
	} else {
		return self.payload.Seek(offset, whence)
	}
}

func (self *responseFile) Close() error {
	self.payload = nil
	return nil
}

func (self *responseFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, fmt.Errorf("readdir() not valid on response objects")
}

func (self *responseFile) Name() string {
	return self.name
}

func (self *responseFile) Size() int64 {
	return self.size
}

func (self *responseFile) Mode() os.FileMode {
	return 0666
}

func (self *responseFile) ModTime() time.Time {
	return time.Now()
}

func (self *responseFile) IsDir() bool {
	return false
}

func (self *responseFile) Sys() interface{} {
	return nil
}

func (self *responseFile) Stat() (os.FileInfo, error) {
	return self, nil
}

type ProxyMount struct {
	MountPoint          string            `json:"mount"`
	URL                 string            `json:"url"`
	Method              string            `json:"method,omitempty"`
	Headers             map[string]string `json:"headers,omitempty"`
	Timeout             time.Duration     `json:"timeout,omitempty"`
	PassthroughRequests bool              `json:"passthrough_requests"`
	PassthroughErrors   bool              `json:"passthrough_errors"`
	Client              *http.Client
}

func (self *ProxyMount) GetMountPoint() string {
	return self.MountPoint
}

func (self *ProxyMount) WillRespondTo(name string, req *http.Request, requestBody io.Reader) bool {
	return strings.HasPrefix(name, self.GetMountPoint())
}

func (self *ProxyMount) OpenWithType(name string, req *http.Request, requestBody io.Reader) (http.File, string, error) {
	var proxyURI string

	if self.Client == nil {
		if self.Timeout == 0 {
			self.Timeout = DefaultProxyMountTimeout
		}

		self.Client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: false,
				},
			},
			Timeout: self.Timeout,
		}
	}

	if self.Method == `` {
		self.Method = `get`
	}

	if req != nil && self.PassthroughRequests {
		if newURL, err := url.Parse(self.URL); err == nil {
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
			return nil, ``, fmt.Errorf("Failed to parse proxy URL: %v", err)
		}
	} else {
		proxyURI = strings.Join([]string{
			strings.TrimSuffix(self.URL, `/`),
			strings.TrimPrefix(name, `/`),
		}, `/`)
	}

	method := strings.ToUpper(self.Method)

	if req != nil && self.PassthroughRequests {
		method = req.Method
	}

	log.Debugf("Proxy URI: %v", proxyURI)

	if newReq, err := http.NewRequest(method, proxyURI, nil); err == nil {
		if req != nil && self.PassthroughRequests {
			for name, values := range req.Header {
				for _, value := range values {
					newReq.Header.Set(name, value)
				}
			}
		}

		for name, value := range self.Headers {
			newReq.Header.Set(name, value)
		}

		if requestBody != nil && self.PassthroughRequests {
			newReq.Body = ioutil.NopCloser(requestBody)
		}

		log.Debugf("ProxyMount: %v %v", newReq.Method, newReq.URL)

		for k, v := range newReq.Header {
			log.Debugf("ProxyMount: [H] %v=%v", k, strings.Join(v, ` `))
		}

		if response, err := self.Client.Do(newReq); err == nil {
			if response.StatusCode < 400 || self.PassthroughErrors {
				if data, err := ioutil.ReadAll(response.Body); err == nil {
					payload := bytes.NewReader(data)

					return &responseFile{
						name:    name,
						size:    payload.Size(),
						payload: payload,
					}, response.Header.Get(`Content-Type`), nil
				} else {
					return nil, ``, err
				}
			} else {
				log.Debugf("ProxyMount: %s %s: %s", method, proxyURI, response.Status)
				return nil, ``, MountHaltErr
			}
		} else {
			return nil, ``, err
		}
	} else {
		return nil, ``, err
	}
}

func (self *ProxyMount) Open(name string) (http.File, error) {
	file, _, err := self.OpenWithType(name, nil, nil)
	return file, err
}
