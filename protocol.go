package diecast

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/typeutil"
)

type Protocol interface {
	Retrieve(*ProtocolRequest) (*ProtocolResponse, error)
}

type ProtocolConfig map[string]any

func (config ProtocolConfig) Get(key string, fallbacks ...any) typeutil.Variant {
	var v = maputil.M(config).Get(key)

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
	Verb              string
	URL               *url.URL
	Binding           *Binding
	Request           *http.Request
	Header            *TemplateHeader
	TemplateData      map[string]any
	TemplateFuncs     FuncMap
	DefaultTimeout    time.Duration
	AdditionalHeaders map[string]any
}

func (config *ProtocolRequest) ReadFile(filename string) ([]byte, error) {
	if b := config.Binding; b != nil {
		if s := b.server; s != nil {
			return readFromFS(s.fs, filename)
		}
	}

	return nil, fmt.Errorf("no such file or directory")
}

func (config *ProtocolRequest) Template(input any) (typeutil.Variant, error) {
	// only do template evaluation if the input is a string that contains "{{" and "}}"
	if vS := typeutil.String(input); strings.Contains(vS, `{{`) && strings.Contains(vS, `}}`) {
		if len(config.TemplateFuncs) > 0 {
			if v, err := EvalInline(vS, config.TemplateData, config.TemplateFuncs); err == nil {
				return typeutil.V(v), nil
			} else {
				return typeutil.V(nil), err
			}
		}
	}

	return typeutil.V(input), nil
}

func (config *ProtocolRequest) Conf(proto string, key string, fallbacks ...any) typeutil.Variant {
	if config.Binding != nil {
		if config.Binding.server != nil {
			if len(config.Binding.server.Protocols) > 0 {
				if cnf, ok := config.Binding.server.Protocols[proto]; ok {
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
	Raw        any
	data       io.ReadCloser
}

func NewProtocolResponse(data io.ReadCloser) *ProtocolResponse {
	return &ProtocolResponse{
		MimeType:   `application/octet-stream`,
		StatusCode: http.StatusOK,
		data:       data,
	}
}

func (config *ProtocolResponse) PeekLen() (int64, error) {
	var buf = bytes.NewBuffer(nil)

	if n, err := io.Copy(buf, config.data); err == nil {
		config.data = io.NopCloser(buf)
		return n, nil
	} else {
		return 0, err
	}
}

func (config *ProtocolResponse) Read(b []byte) (int, error) {
	if config.data == nil {
		return 0, io.EOF
	} else {
		return config.data.Read(b)
	}
}

func (config *ProtocolResponse) Close() error {
	if config.data == nil {
		return nil
	} else {
		return config.data.Close()
	}
}
