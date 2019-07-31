package diecast

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"net/url"

	"github.com/PuerkitoBio/goquery"
	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/typeutil"
	"github.com/ghodss/yaml"
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
	if len(self.TemplateFuncs) > 0 {
		return typeutil.V(
			MustEvalInline(typeutil.String(input), self.TemplateData, self.TemplateFuncs),
		)
	} else {
		return typeutil.V(input)
	}
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

func (self *ProtocolResponse) Decode() (interface{}, error) {
	return self.decode()
}

func (self *ProtocolResponse) DecodeAs(parser string) (interface{}, error) {
	return self.decode(parser)
}

func (self *ProtocolResponse) decode(forceParser ...string) (interface{}, error) {
	var parser string

	if len(forceParser) > 0 && forceParser[0] != `` {
		parser = forceParser[0]
	}

	if data, err := ioutil.ReadAll(self.data); err == nil && self.StatusCode < 400 {
		mimeType, _, _ := mime.ParseMediaType(self.MimeType)

		if mimeType == `` {
			mimeType, _ = stringutil.SplitPair(self.MimeType, `;`)
		}

		// only do response body processing if there is data to process
		if len(data) > 0 {
			if parser == `` {
				switch mimeType {
				case `application/json`:
					parser = `json`
				case `application/x-yaml`, `application/yaml`, `text/yaml`:
					parser = `yaml`
				case `text/html`:
					parser = `html`
				case `text/xml`:
					parser = `xml`
				}
			}

			switch parser {
			case `json`, ``:
				// if the parser is unset, and the response type is NOT application/json, then
				// just read the response as plain text and return it.
				//
				// If you're certain the response actually is JSON, then explicitly set Parser==`json`
				//
				if parser == `` && mimeType != `application/json` {
					return string(data), nil
				} else {
					var rv interface{}

					if err := json.Unmarshal(data, &rv); err == nil {
						return rv, nil
					} else {
						return nil, err
					}
				}

			case `yaml`:
				var rv interface{}
				if err := yaml.Unmarshal(data, &rv); err == nil {
					return rv, nil
				} else {
					return nil, err
				}

			case `html`:
				return goquery.NewDocumentFromReader(bytes.NewBuffer(data))

			case `tsv`:
				return xsvToArray(data, '\t')

			case `csv`:
				return xsvToArray(data, ',')

			case `xml`:
				return xmlToMap(data)

			case `text`:
				return string(data), nil

			case `raw`:
				return template.HTML(string(data)), nil

			default:
				return nil, fmt.Errorf("[%s] Unknown response parser %q", id, parser)
			}
		} else {
			return nil, nil
		}
	} else {
		return nil, err
	}
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
