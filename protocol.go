package diecast

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/typeutil"
)

type Protocol interface {
	Retrieve(*ProtocolRequest) (*ProtocolResponse, error)
}

type ProtocolConfig map[string]interface{}

func (self ProtocolConfig) Get(key string, fallbacks ...interface{}) typeutil.Variant {
	v := maputil.M(self).Get(key)

	if v.IsNil() {
		if len(fallbacks) > 0 {
			return typeutil.V(fallbacks[0])
		} else {
			return typeutil.V(nil)
		}
	} else {
		return v
	}
}

type ProtocolRequest struct {
	Verb          string
	URL           *url.URL
	Binding       *Binding
	Request       *http.Request
	Header        *TemplateHeader
	TemplateData  map[string]interface{}
	TemplateFuncs FuncMap
}

func (self *ProtocolRequest) Template(input interface{}) typeutil.Variant {
	// only do template evaluation if the input is a string that contains "{{" and "}}"
	if vS := typeutil.String(input); strings.Contains(vS, `{{`) && strings.Contains(vS, `}}`) {
		if len(self.TemplateFuncs) > 0 {
			return typeutil.V(
				MustEvalInline(typeutil.String(input), self.TemplateData, self.TemplateFuncs),
			)
		}
	}

	return typeutil.V(input)
}

func (self *ProtocolRequest) Conf(proto string, key string, fallbacks ...interface{}) typeutil.Variant {
	if self.Binding != nil {
		if self.Binding.server != nil {
			if len(self.Binding.server.Protocols) > 0 {
				if cnf, ok := self.Binding.server.Protocols[proto]; ok {
					return cnf.Get(key, fallbacks...)
				}
			}
		}
	}

	if len(fallbacks) > 0 {
		return typeutil.V(fallbacks[0])
	} else {
		return typeutil.V(nil)
	}
}

type ProtocolResponse struct {
	MimeType   string
	StatusCode int
	Raw        interface{}
	data       io.ReadCloser
}

func (self *ProtocolResponse) PeekLen() (int64, error) {
	buf := bytes.NewBuffer(nil)

	if n, err := io.Copy(buf, self.data); err == nil {
		self.data = ioutil.NopCloser(buf)
		return n, nil
	} else {
		return 0, err
	}
}

func (self *ProtocolResponse) Read(b []byte) (int, error) {
	if self.data == nil {
		return 0, io.EOF
	} else {
		return self.data.Read(b)
	}
}

func (self *ProtocolResponse) Close() error {
	if self.data == nil {
		return nil
	} else {
		return self.data.Close()
	}
}
