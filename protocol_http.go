package diecast

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
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

// The HTTP binding protocol is used to interact with web servers and RESTful APIs.
// It is specified with URLs that use the http:// or https:// schemes.
//
type HttpProtocol struct {
}

func (self *HttpProtocol) Retrieve(rr *ProtocolRequest) (*ProtocolResponse, error) {
	var id = reqid(rr.Request)
	var trueHost string

	// Okay...so...
	//
	// We want the ability to support requests to vanilla HTTP(S) connections exposed via local Unix
	// sockets.  Conventionally, HTTP URL schemes don't really support this, but the "protocol+transport"
	// convention allows for us to kinda fake it a bit.
	//
	// However, the path to the local Unix socket is *not* guaranteed to be a valid hostname.  So we're
	// going to put that value into the request context and use "localhost" just to pass muster when
	// parsing the URL.  The http.RoundTripper used below will be looking for this context value.
	//
	switch rr.URL.Scheme {
	case `http+unix`, `https+unix`:
		rr.URL.Scheme = strings.TrimSuffix(rr.URL.Scheme, `+unix`)
		trueHost = strings.ReplaceAll(rr.URL.Host, weirdPathsInHostnamesPlaceholder, `/`)
		rr.URL.Host = `localhost`
	}

	if request, err := http.NewRequest(rr.Verb, rr.URL.String(), nil); err == nil {
		httputil.RequestSetValue(request, `diecastSocketPath`, trueHost)

		// build request querystring
		// ---------------------------------------------------------------------------------------------

		// eval and add query string parameters to request
		var qs = request.URL.Query()

		for k, v := range rr.Binding.Params {
			var vS string

			if typeutil.IsArray(v) {
				var joiner = DefaultParamJoiner

				if j := rr.Binding.ParamJoiner; j != `` {
					joiner = j
				}

				vS = strings.Join(sliceutil.Stringify(v), joiner)
			} else {
				vS = stringutil.MustString(v)
			}

			if !rr.Binding.NoTemplate {
				if p, err := rr.Template(vS); err == nil {
					vS = p.String()
				} else {
					return nil, fmt.Errorf("param %q: %v", k, err)
				}
			}

			log.Debugf("[%s]  binding %q: param %v=%v", id, rr.Binding.Name, k, vS)
			qs.Set(k, vS)
		}

		request.URL.RawQuery = qs.Encode()

		// build request body
		// ---------------------------------------------------------------------------------------------
		// binding body content can be specified either as key-value pairs encoded using a
		// set of pre-defined encoders, or as a raw string (Content-Type can be explicitly set
		// via Headers).
		//
		var body bytes.Buffer

		if rr.Binding.BodyParams != nil {
			var bodyParams = make(map[string]interface{})

			if len(rr.Binding.BodyParams) > 0 {
				// evaluate each body param value as a template (unless explicitly told not to)
				if err := maputil.Walk(rr.Binding.BodyParams, func(value interface{}, path []string, isLeaf bool) error {
					if isLeaf {
						if !rr.Binding.NoTemplate {
							var rendered typeutil.Variant

							if r, err := rr.Template(value); err == nil {
								rendered = r
							} else {
								return fmt.Errorf("body param: %v", err)
							}

							if typeutil.IsScalar(rendered.Value) {
								value = rendered.Auto()
							} else {
								value = rendered.Value
							}
						}

						maputil.DeepSet(bodyParams, path, stringutil.Autotype(value))
					}

					return nil
				}); err == nil {
					log.Debugf("[%s]  binding %q: bodyparam %#v", id, rr.Binding.Name, bodyParams)
				} else {
					return nil, err
				}
			}

			// perform encoding of body data
			if len(bodyParams) > 0 {
				switch rr.Binding.Formatter {
				case `json`, ``:
					// JSON-encode params into the body buffer
					if err := json.NewEncoder(&body).Encode(&bodyParams); err != nil {
						return nil, err
					}

					// set body and content type
					request.Body = ioutil.NopCloser(&body)
					request.Header.Set(`Content-Type`, `application/json`)

				case `form`:
					var form = url.Values{}

					// add params to form values
					for k, v := range bodyParams {
						form.Add(k, fmt.Sprintf("%v", v))
					}

					// write encoded form values to body buffer
					if _, err := body.WriteString(form.Encode()); err != nil {
						return nil, err
					}

					// set body and content type
					request.Body = ioutil.NopCloser(&body)
					request.Header.Set(`Content-Type`, `application/x-www-form-urlencoded`)

				default:
					return nil, fmt.Errorf("[%s] Unknown request formatter %q", id, rr.Binding.Formatter)
				}
			}
		} else if rr.Binding.RawBody != `` {
			var payload []byte

			if b, err := rr.Template(rr.Binding.RawBody); err == nil {
				payload = b.Bytes()
			} else {
				return nil, fmt.Errorf("rawbody: %v", err)
			}

			log.Debugf("[%s]  binding %q: rawbody (%d bytes)", id, rr.Binding.Name, len(payload))
			request.Body = ioutil.NopCloser(bytes.NewBuffer(payload))
		}

		// build request headers
		// ---------------------------------------------------------------------------------------------

		// if specified, have the binding request inherit the headers from the initiating request
		if !rr.Binding.SkipInheritHeaders {
			for k, _ := range rr.Request.Header {
				var v = rr.Request.Header.Get(k)
				log.Debugf("[%s]  binding %q: inherit %v=%v", id, rr.Binding.Name, k, v)
				request.Header.Set(k, v)
			}
		}

		// add headers to request
		for k, v := range rr.Binding.Headers {
			if !rr.Binding.NoTemplate {
				if vv, err := rr.Template(v); err == nil {
					v = vv.String()
				} else {
					return nil, fmt.Errorf("headers: %v", err)
				}
			}

			log.Debugf("[%s]  binding %q:  header %v=%v", id, rr.Binding.Name, k, v)
			request.Header.Add(k, v)
		}

		// finally, tack on any request-specific headers
		if len(rr.AdditionalHeaders) > 0 {
			for k, v := range rr.AdditionalHeaders {
				request.Header.Add(k, typeutil.String(v))
			}
		}

		request.Header.Set(`X-Diecast-Binding`, rr.Binding.Name)

		// big block of custom TLS override setup
		// ---------------------------------------------------------------------------------------------
		var newTCC = &tls.Config{
			InsecureSkipVerify: rr.Binding.Insecure,
			RootCAs:            rr.Binding.server.altRootCaPool,
		}

		// jump through some hoops to allow per-binding TLS client auth
		if crt := rr.Binding.TlsCertificate; crt != `` {
			if key := rr.Binding.TlsKey; key != `` {
				if crtdata, err := rr.ReadFile(crt); err == nil {
					if keydata, err := rr.ReadFile(key); err == nil {
						if certificate, err := tls.X509KeyPair(crtdata, keydata); err == nil {
							newTCC.GetClientCertificate = func(info *tls.CertificateRequestInfo) (*tls.Certificate, error) {
								return &certificate, nil
							}
						} else {
							return nil, fmt.Errorf("bad certificate: %v", err)
						}
					} else {
						return nil, fmt.Errorf("bad tls key: %v", err)
					}
				} else {
					return nil, fmt.Errorf("bad tls cert: %v", err)
				}
			}
		}

		newTCC.BuildNameToCertificate()

		if transport, ok := BindingClient.Transport.(*http.Transport); ok {
			if tcc := transport.TLSClientConfig; tcc != nil {
				tcc.InsecureSkipVerify = newTCC.InsecureSkipVerify
				tcc.RootCAs = newTCC.RootCAs
			} else {
				transport.TLSClientConfig = newTCC
			}
		} else {
			BindingClient.Transport = &http.Transport{
				TLSClientConfig: newTCC,
			}
		}

		if timeout := typeutil.V(rr.Binding.Timeout).Duration(); timeout > 0 {
			if timeout < time.Microsecond {
				// probably given as numeric seconds
				timeout = timeout * time.Second
			} else if timeout < time.Millisecond {
				// probably given as numeric seconds
				timeout = timeout * time.Millisecond
			}

			BindingClient.Timeout = timeout
		}

		if BindingClient.Timeout == 0 {
			BindingClient.Timeout = DefaultBindingTimeout
		}

		log.Debugf("[%s]  binding: timeout=%v", id, BindingClient.Timeout)

		if request.URL.Scheme == `https` && rr.Binding.Insecure {
			log.Noticef("[%s] SSL/TLS certificate validation is disabled for this request.", id)
			log.Noticef("[%s] This is insecure as the response can be tampered with.", id)
		}

		// end TLS setup
		// ---------------------------------------------------------------------------------------------

		if t, ok := BindingClient.Transport.(*http.Transport); ok {
			BindingClient.Transport = &transportAwareRoundTripper{
				transport: t,
			}
		}

		// tell the server we want to close the connection when done
		request.Close = true

		// perform binding request
		// ---------------------------------------------------------------------------------------------
		if res, err := BindingClient.Do(request); err == nil {
			log.Infof("[%s] Binding: < HTTP %d (body: %d bytes)", id, res.StatusCode, res.ContentLength)

			// debug log response headers
			for k, v := range res.Header {
				log.Debugf("[%s]  [H] %v: %v", id, k, strings.Join(v, ` `))
			}

			// stub out the response
			var response = &ProtocolResponse{
				Raw:        res,
				StatusCode: res.StatusCode,
			}

			// work out Content-Type
			if mt, _, err := mime.ParseMediaType(res.Header.Get(`Content-Type`)); err == nil {
				response.MimeType = mt
			} else {
				response.MimeType = res.Header.Get(`Content-Type`)
			}

			// decode the response body (e.g.: get a stream of bytes out of compressed responses)
			if res.Body != nil {
				if body, err := httputil.DecodeResponse(res); err == nil {
					if rc, ok := body.(io.ReadCloser); ok {
						response.data = rc
					} else {
						response.data = ioutil.NopCloser(body)
					}
				} else {
					return nil, err
				}
			}

			return response, nil
		} else {
			if res != nil && res.StatusCode > 0 {
				log.Warningf("[%s] Binding: < HTTP %d (body: %d bytes)", id, res.StatusCode, res.ContentLength)
			} else {
				log.Warningf("[%s] Binding: < error: %v", id, err)
			}

			return nil, err
		}
	} else {
		return nil, fmt.Errorf("[%s] request: %v", id, err)
	}
}
