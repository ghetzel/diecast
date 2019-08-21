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
	id := reqid(rr.Request)

	if request, err := http.NewRequest(rr.Verb, rr.URL.String(), nil); err == nil {
		// build request querystring
		// -------------------------------------------------------------------------------------

		// eval and add query string parameters to request
		qs := request.URL.Query()

		for k, v := range rr.Binding.Params {
			var vS string

			if typeutil.IsArray(v) {
				joiner := DefaultParamJoiner

				if j := rr.Binding.ParamJoiner; j != `` {
					joiner = j
				}

				vS = strings.Join(sliceutil.Stringify(v), joiner)
			} else {
				vS = stringutil.MustString(v)
			}

			if !rr.Binding.NoTemplate {
				vS = rr.Template(vS).String()
			}

			log.Debugf("[%s]  binding %q: param %v=%v", id, rr.Binding.Name, k, vS)
			qs.Set(k, vS)
		}

		request.URL.RawQuery = qs.Encode()

		// build request body
		// -------------------------------------------------------------------------------------
		// binding body content can be specified either as key-value pairs encoded using a
		// set of pre-defined encoders, or as a raw string (Content-Type can be explicitly set
		// via Headers).
		//
		var body bytes.Buffer

		if rr.Binding.BodyParams != nil {
			bodyParams := make(map[string]interface{})

			if len(rr.Binding.BodyParams) > 0 {
				// evaluate each body param value as a template (unless explicitly told not to)
				if err := maputil.Walk(rr.Binding.BodyParams, func(value interface{}, path []string, isLeaf bool) error {
					if isLeaf {
						if !rr.Binding.NoTemplate {
							value = rr.Template(value).Auto()
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
					form := url.Values{}

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
			payload := rr.Template(rr.Binding.RawBody).Bytes()
			log.Debugf("[%s]  binding %q: rawbody (%d bytes)", id, rr.Binding.Name, len(payload))
			request.Body = ioutil.NopCloser(bytes.NewBuffer(payload))
		}

		// build request headers
		// -------------------------------------------------------------------------------------

		// if specified, have the binding request inherit the headers from the initiating request
		if !rr.Binding.SkipInheritHeaders {
			for k, _ := range rr.Request.Header {
				v := rr.Request.Header.Get(k)
				log.Debugf("[%s]  binding %q: inherit %v=%v", id, rr.Binding.Name, k, v)
				request.Header.Set(k, v)
			}
		}

		// add headers to request
		for k, v := range rr.Binding.Headers {
			if !rr.Binding.NoTemplate {
				v = rr.Template(v).String()
			}

			log.Debugf("[%s]  binding %q:  header %v=%v", id, rr.Binding.Name, k, v)
			request.Header.Add(k, v)
		}

		request.Header.Set(`X-Diecast-Binding`, rr.Binding.Name)

		// big block of custom TLS override setup
		// -------------------------------------------------------------------------------------
		newTCC := &tls.Config{
			InsecureSkipVerify: rr.Binding.Insecure,
			RootCAs:            rr.Binding.server.altRootCaPool,
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
		// -------------------------------------------------------------------------------------

		// tell the server we want to close the connection when done
		request.Close = true

		// perform binding request
		// -------------------------------------------------------------------------------------
		if res, err := BindingClient.Do(request); err == nil {
			log.Infof("[%s] Binding: < HTTP %d (body: %d bytes)", id, res.StatusCode, res.ContentLength)

			// debug log response headers
			for k, v := range res.Header {
				log.Debugf("[%s]  [H] %v: %v", id, k, strings.Join(v, ` `))
			}

			// stub out the response
			response := &ProtocolResponse{
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
