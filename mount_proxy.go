package diecast

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
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
	opentracing "github.com/opentracing/opentracing-go"
)

var DefaultProxyMountTimeout = time.Duration(10) * time.Second
var MaxBufferedBodySize int64 = 16535

type ProxyMount struct {
	MountPoint              string         `json:"-"`
	URL                     string         `json:"-"`
	Method                  string         `json:"method,omitempty"`
	Headers                 map[string]any `json:"headers,omitempty"`
	ResponseHeaders         map[string]any `json:"response_headers,omitempty"`
	ResponseCode            int            `json:"response_code"`
	RedirectOnSuccess       string         `json:"redirect_on_success"`
	Params                  map[string]any `json:"params,omitempty"`
	Timeout                 any            `json:"timeout,omitempty"`
	PassthroughRequests     bool           `json:"passthrough_requests"`
	PassthroughHeaders      bool           `json:"passthrough_headers"`
	PassthroughQueryStrings bool           `json:"passthrough_query_strings"`
	PassthroughBody         bool           `json:"passthrough_body"`
	PassthroughErrors       bool           `json:"passthrough_errors"`
	PassthroughRedirects    bool           `json:"passthrough_redirects"`
	PassthroughUserAgent    bool           `json:"passthrough_user_agent"`
	StripPathPrefix         string         `json:"strip_path_prefix"`
	AppendPathPrefix        string         `json:"append_path_prefix"`
	Insecure                bool           `json:"insecure"`
	BodyBufferSize          int64          `json:"body_buffer_size"`
	CloseConnection         *bool          `json:"close_connection"`
	Client                  *http.Client
	urlRewriteFrom          string
	urlRewriteTo            string
}

func (mount *ProxyMount) GetMountPoint() string {
	return mount.MountPoint
}

func (mount *ProxyMount) GetTarget() string {
	return mount.URL
}

func (mount *ProxyMount) WillRespondTo(name string, req *http.Request, requestBody io.Reader) bool {
	return strings.HasPrefix(name, mount.GetMountPoint())
}

func (mount *ProxyMount) OpenWithType(name string, req *http.Request, requestBody io.Reader) (res *MountResponse, err error) {
	var tracer opentracing.Tracer
	var spanopts []opentracing.StartSpanOption
	var childSpan opentracing.Span

	// if the originating request has a tracing span, we're going to create a child span of that
	// to trace this binding evaluation
	if parentSpan, ok := httputil.RequestGetValue(req, JaegerSpanKey).Value.(opentracing.Span); ok {
		tracer = opentracing.GlobalTracer()
		spanopts = append(spanopts, opentracing.ChildOf(parentSpan.Context()))
		spanopts = append(spanopts, opentracing.Tag{
			Key:   `diecast.mount`,
			Value: mount.MountPoint,
		})
	} else {
		tracer = new(opentracing.NoopTracer)
	}

	var traceName = fmt.Sprintf("Mount: %s", mount.MountPoint)
	var traceHeaders = make(http.Header)

	childSpan = tracer.StartSpan(traceName, spanopts...)
	childSpan.Tracer().Inject(
		childSpan.Context(),
		opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(traceHeaders),
	)

	res, err = mount.openWithType(name, req, requestBody, traceHeaders)

	if err == nil {
		if res.RedirectTo != `` {
			childSpan.SetTag(`http.status_code`, res.RedirectCode)
			childSpan.SetTag(`http.redirect`, res.RedirectTo)
		} else {
			childSpan.SetTag(`http.status_code`, res.StatusCode)
		}
	} else {
		childSpan.SetTag(`error`, err.Error())
	}

	childSpan.Finish()
	return
}

func (mount *ProxyMount) openWithType(name string, req *http.Request, requestBody io.Reader, _ http.Header) (*MountResponse, error) {
	var id = reqid(req)
	var proxyURI string
	var timeout time.Duration

	if mount.Client == nil {
		if t, ok := mount.Timeout.(string); ok {
			if tm, err := time.ParseDuration(t); err == nil {
				timeout = tm
			} else {
				log.Warningf("[%s] proxy: INVALID TIMEOUT %q (%v), using default", id, t, err)
				timeout = DefaultProxyMountTimeout
			}
		} else if tm, ok := mount.Timeout.(time.Duration); ok {
			timeout = tm
		}

		if timeout == 0 {
			timeout = DefaultProxyMountTimeout
		}

		mount.Client = &http.Client{
			// TODO: things and stuff to make this use *transportAwareRoundTripper
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: mount.Insecure,
				},
			},
			Timeout: timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if mount.PassthroughRedirects {
					return http.ErrUseLastResponse
				} else if mount.urlRewriteTo == `` {
					if len(via) > 0 {
						mount.urlRewriteFrom = via[len(via)-1].URL.String()
						mount.urlRewriteFrom = strings.TrimSuffix(mount.urlRewriteFrom, `/`)
						mount.urlRewriteTo = req.URL.String()
					}
				}

				return nil
			},
		}
	}

	if req != nil && (mount.PassthroughRequests || mount.PassthroughQueryStrings) {
		if newURL, err := url.Parse(mount.url()); err == nil {
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
			return nil, fmt.Errorf("failed to parse proxy URL: %v", err)
		}
	} else {
		proxyURI = strings.Join([]string{
			strings.TrimSuffix(mount.url(), `/`),
			strings.TrimPrefix(name, `/`),
		}, `/`)
	}

	var method = strings.ToUpper(mount.Method)

	if method == `` {
		if req != nil {
			method = req.Method
		} else {
			method = `GET`
		}
	}

	if newReq, err := http.NewRequest(method, proxyURI, nil); err == nil {
		if pp := mount.StripPathPrefix; pp != `` {
			newReq.URL.Path = strings.TrimPrefix(newReq.URL.Path, pp)
		}

		if pp := mount.AppendPathPrefix; pp != `` {
			newReq.URL.Path = pp + newReq.URL.Path
		}

		if req != nil && (mount.PassthroughRequests || mount.PassthroughHeaders) {
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
		if !mount.PassthroughUserAgent {
			newReq.Header.Set(`User-Agent`, DiecastUserAgentString)
		}

		// option to control whether we tell the remote server to close the connection
		if mount.CloseConnection != nil {
			if c := *mount.CloseConnection; c {
				newReq.Header.Set(`Connection`, `close`)
			} else {
				newReq.Header.Set(`Connection`, `keep-alive`)
			}
		}

		// add explicit headers to new request
		for name, value := range mount.Headers {
			newReq.Header.Set(name, typeutil.String(value))
		}

		newReq.Header.Set(`Accept-Encoding`, `identity`)

		// inject params into new request
		for name, value := range mount.Params {
			if newReq.URL.Query().Get(name) == `` {
				log.Debugf("[%s] proxy: [Q] %v=%v", id, name, value)
				httputil.SetQ(newReq.URL, name, value)
			}
		}

		if requestBody != nil && (mount.PassthroughRequests || mount.PassthroughBody) {
			var buf bytes.Buffer
			var bufsz int64 = MaxBufferedBodySize

			if mount.BodyBufferSize > 0 {
				bufsz = mount.BodyBufferSize
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
		response, err := mount.Client.Do(newReq)
		log.Debugf("[%s] proxy: responded in %v", id, time.Since(reqStartAt))

		if err == nil {
			if response.Body != nil {
				defer response.Body.Close()
			}

			// add explicit response headers to response
			for name, value := range mount.ResponseHeaders {
				response.Header.Set(name, typeutil.String(value))
			}

			// override the response status code (if specified)
			if mount.ResponseCode > 0 {
				response.StatusCode = mount.ResponseCode
			}

			// provide a header redirect if so requested
			if response.StatusCode < 400 && mount.RedirectOnSuccess != `` {
				if response.StatusCode < 300 {
					response.StatusCode = http.StatusTemporaryRedirect
				}

				response.Header.Set(`Location`, mount.RedirectOnSuccess)
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

			if response.StatusCode < 400 || mount.PassthroughErrors {
				var responseBody io.Reader

				if body, err := httputil.DecodeResponse(response); err == nil {
					responseBody = body

					// whatever the encoding was before, it's definitely "identity" now
					response.Header.Set(`Content-Encoding`, `identity`)
				} else {
					return nil, err
				}

				if data, err := io.ReadAll(responseBody); err == nil {
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
				if data, err := io.ReadAll(response.Body); err == nil {
					for _, line := range stringutil.SplitLines(data, "\n") {
						log.Debugf("[%s] proxy: [B] %s", id, line)
					}
				}

				log.Debugf("[%s] proxy: %s %s: %s", id, method, newReq.URL, response.Status)

				return nil, ErrMountHalt
			}
		} else {
			return nil, err
		}
	} else {
		return nil, err
	}
}

func (mount *ProxyMount) url() string {
	var uri = mount.URL

	if from := mount.urlRewriteFrom; from != `` {
		if to := mount.urlRewriteTo; to != `` {
			uri = strings.Replace(uri, from, to, 1)

			log.Debugf("Rewriting %v to %v due to earlier redirect", mount.urlRewriteFrom, mount.urlRewriteTo)
		}
	}

	return uri
}

func (mount *ProxyMount) String() string {
	return fmt.Sprintf(
		"%v -> %v %v (passthrough requests=%v errors=%v)",
		mount.MountPoint,
		strings.ToUpper(sliceutil.OrString(mount.Method, `<method>`)),
		mount.url(),
		mount.PassthroughRequests,
		mount.PassthroughErrors,
	)
}

func (mount *ProxyMount) Open(name string) (http.File, error) {
	return openAsHttpFile(mount, name)
}
