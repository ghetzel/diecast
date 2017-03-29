package diecast

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
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
	MountPoint string            `json:"mount"`
	URL        string            `json:"url"`
	Method     string            `json:"method,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
	Timeout    time.Duration     `json:"timeout,omitempty"`
	Client     *http.Client
}

func (self *ProxyMount) GetMountPoint() string {
	return self.MountPoint
}

func (self *ProxyMount) WillRespondTo(name string) bool {
	return strings.HasPrefix(name, self.GetMountPoint())
}

func (self *ProxyMount) OpenWithType(name string) (http.File, string, error) {
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

	url := strings.Join([]string{
		strings.TrimSuffix(self.URL, `/`),
		strings.TrimPrefix(name, `/`),
	}, `/`)

	method := strings.ToUpper(self.Method)

	log.Debugf("url: %v", url)

	if req, err := http.NewRequest(method, url, nil); err == nil {
		for name, value := range self.Headers {
			req.Header.Set(name, value)
		}

		if response, err := self.Client.Do(req); err == nil {
			if response.StatusCode < 400 {
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
				log.Debugf("ProxyMount: %s %s: %s", method, url, response.Status)
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
	file, _, err := self.OpenWithType(name)
	return file, err
}
