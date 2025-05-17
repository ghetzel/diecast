package diecast

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/ghetzel/go-stockutil/typeutil"
)

func kvValues(kv []KV) (values []any) {
	for _, item := range kv {
		values = append(values, item.V)
	}

	return
}

type KV struct {
	K string `json:"key"`
	V any    `json:"value"`
}

type Cookie struct {
	Name     string         `yaml:"name,omitempty"     json:"name,omitempty"`
	Value    any            `yaml:"-"                  json:"value,omitempty"`
	Path     string         `yaml:"path,omitempty"     json:"path,omitempty"`
	Domain   string         `yaml:"domain,omitempty"   json:"domain,omitempty"`
	MaxAge   *int           `yaml:"maxAge,omitempty"   json:"maxAge,omitempty"`
	Secure   *bool          `yaml:"secure,omitempty"   json:"secure,omitempty"`
	HttpOnly *bool          `yaml:"httpOnly,omitempty" json:"httpOnly,omitempty"`
	SameSite CookieSameSite `yaml:"sameSite,omitempty" json:"sameSite,omitempty"`
}

type CookieSameSite string

const (
	SameSiteDefault CookieSameSite = ``
	SameSiteLax     CookieSameSite = `lax`
	SameSiteStrict  CookieSameSite = `strict`
	SameSiteNone    CookieSameSite = `none`
)

func MakeCookieSameSite(sameSite http.SameSite) CookieSameSite {
	switch sameSite {
	case http.SameSiteDefaultMode:
		return SameSiteDefault
	case http.SameSiteLaxMode:
		return SameSiteLax
	case http.SameSiteStrictMode:
		return SameSiteStrict
	default:
		return SameSiteNone
	}
}

func (info CookieSameSite) SameSite() http.SameSite {
	switch info {
	case SameSiteLax:
		return http.SameSiteLaxMode
	case SameSiteStrict:
		return http.SameSiteStrictMode
	// case SameSiteNone:
	// 	return http.SameSiteNoneMode
	default:
		return http.SameSiteDefaultMode
	}
}

type RequestTlsCertName struct {
	SerialNumber       string `json:"serialnumber"`
	CommonName         string `json:"common"`
	Country            string `json:"country"`
	Organization       string `json:"organization"`
	OrganizationalUnit string `json:"orgunit"`
	Locality           string `json:"locality"`
	State              string `json:"state"`
	StreetAddress      string `json:"street"`
	PostalCode         string `json:"postalcode"`
}

type RequestTlsCertSan struct {
	DNSNames       []string `json:"dns"`
	EmailAddresses []string `json:"email"`
	IPAddresses    []string `json:"ip"`
	URIs           []string `json:"uri"`
}

type RequestTlsCertInfo struct {
	Issuer                 RequestTlsCertName `json:"issuer"`
	Subject                RequestTlsCertName `json:"subject"`
	NotBefore              time.Time          `json:"not_before"`
	NotAfter               time.Time          `json:"not_after"`
	SecondsLeft            int                `json:"seconds_left"`
	OcspServer             []string           `json:"ocsp_server"`
	IssuingCertUrl         []string           `json:"issuing_cert_url"`
	Version                int                `json:"version"`
	SerialNumber           string             `json:"serialnumber"`
	SubjectAlternativeName *RequestTlsCertSan `json:"san"`
}

type RequestTlsInfo struct {
	Version                    string               `json:"version"`
	HandshakeComplete          bool                 `json:"handshake_complete"`
	DidResume                  bool                 `json:"did_resume"`
	CipherSuite                string               `json:"cipher_suite"`
	NegotiatedProtocol         string               `json:"negotiated_protocol"`
	NegotiatedProtocolIsMutual bool                 `json:"negotiated_protocol_is_mutual"`
	ServerName                 string               `json:"server_name"`
	TlsUnique                  []byte               `json:"tls_unique"`
	Client                     RequestTlsCertInfo   `json:"client"`
	ClientChain                []RequestTlsCertInfo `json:"client_chain"`
}

type RequestUrlInfo struct {
	Unmodified string         `json:"unmodified"`
	String     string         `json:"string"`
	Scheme     string         `json:"scheme"`
	Host       string         `json:"host"`
	Port       int            `json:"port"`
	Path       string         `json:"path"`
	Fragment   string         `json:"fragment"`
	Query      map[string]any `json:"query"`
	Params     []KV           `json:"params"`
}

func (info RequestUrlInfo) ParamsSlice() []any {
	var params []any

	for _, kv := range info.Params {
		params = append(params, kv.V)
	}

	return params
}

type RequestBody struct {
	Length   int64         `json:"size"`
	Raw      []byte        `json:"raw"`
	String   string        `json:"string"`
	Parsed   any           `json:"parsed"`
	Error    string        `json:"error,omitempty"`
	Loaded   bool          `json:"loaded,omitempty"`
	Original io.ReadCloser `json:"-"`
	current  *bytes.Buffer
}

func (info *RequestBody) Read(b []byte) (int, error) {
	if info.current == nil {
		info.current = bytes.NewBuffer(info.Raw)
	}

	return info.current.Read(b)
}

func (info *RequestBody) Close() error {
	info.current = nil
	return nil
}

type RequestInfo struct {
	ID               string            `json:"id"`
	Timestamp        int64             `json:"timestamp"`
	Method           string            `json:"method"`
	Protocol         string            `json:"protocol"`
	ContentLength    int64             `json:"length"`
	TransferEncoding []string          `json:"encoding"`
	Headers          map[string]any    `json:"headers"`
	Cookies          map[string]Cookie `json:"cookies"`
	RemoteIP         string            `json:"remote_ip"`
	RemotePort       int               `json:"remote_port"`
	RemoteAddr       string            `json:"remote_address"`
	Host             string            `json:"host"`
	URL              RequestUrlInfo    `json:"url"`
	TLS              *RequestTlsInfo   `json:"tls"`
	CSRFToken        string            `json:"csrftoken,omitempty"`
	Body             *RequestBody      `json:"body,omitempty"`
}

func (info *RequestInfo) asMap() (map[string]any, error) {
	var rv map[string]any

	if data, err := json.Marshal(info); err == nil {
		if err := json.Unmarshal(data, &rv); err == nil {
			return rv, nil
		} else {
			return nil, err
		}
	} else {
		return nil, err
	}
}

func (info *RequestInfo) Header(key string) typeutil.Variant {
	if v, ok := info.Headers[key]; ok {
		return typeutil.V(v)
	}

	return typeutil.V(nil)
}

func (info *RequestInfo) Cookie(key string) *Cookie {
	if c, ok := info.Cookies[key]; ok {
		return &c
	}

	return nil
}
