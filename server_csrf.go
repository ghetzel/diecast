package diecast

import (
	"bytes"
	"crypto/rand"
	"crypto/subtle"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/ghetzel/go-stockutil/httputil"
	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/typeutil"
)

const DefaultCsrfInjectFormFieldSelector = `form[method="post"], form[method="POST"], form[method="Post"]` // if you need more case permutations than this, you may override this default
const DefaultCsrfInjectFieldFormat = `<input type="hidden" name="%s" value="%s">`
const CsrfTokenLength = 32
const ContextCsrfToken = `csrf-token`
const ContextStatusKey = `response-status-code`
const ContextErrorKey = `response-error-message`

var DefaultCsrfHeaderName = `X-CSRF-Token`
var DefaultCsrfFormFieldName = `csrf_token`
var DefaultCsrfCookieName = `csrf_token`

var DefaultCsrfInjectMediaTypes = []string{
	`text/html`,
}

type CsrfMethod string

const (
	DoubleSubmitCookie CsrfMethod = `cookie`
	HMAC                          = `hmac`
)

type CSRF struct {
	Enable                  bool     `yaml:"enable"                  json:"enable"`                  // Whether to enable stateless CSRF protection
	Except                  []string `yaml:"except"                  json:"except"`                  // A list of paths and path globs that should not be covered by CSRF protection
	Cookie                  *Cookie  `yaml:"cookie"                  json:"cookie"`                  // Specify default fields for the CSRF cookie that is set
	HeaderName              string   `yaml:"header"                  json:"header"`                  // The name of the HTTP header that CSRF tokens may be present in (default: X-CSRF-Token)
	FormFieldName           string   `yaml:"field"                   json:"field"`                   // The name of the HTML form fieldthat CSRF tokens may be present in (default: csrf_token)
	InjectFormFields        bool     `yaml:"injectFormFields"        json:"injectFormFields"`        // If true, a postprocessor will be added that injects a hidden <input> field into all <form> elements returned from Diecast
	InjectFormFieldSelector string   `yaml:"injectFormFieldSelector" json:"injectFormFieldSelector"` // A CSS selector used to locate <form> tags that need the CSRF <input> field injected.
	InjectFormFieldTemplate string   `yaml:"injectFormFieldTemplate" json:"injectFormFieldTemplate"` // Specify the format string that will be used to replace </form> tags with the injected field.
	InjectableMediaTypes    []string `yaml:"injectableMediaTypes"    json:"injectableMediaTypes"`    // Specify a list of Media Types (e.g.: MIME or Content-Types) that will have injection attempted on them (if enabled)
	server                  *Server
	registered              bool
	// Method                  CsrfMethod `yaml:"method"                  json:"method"`                  // Specify the method to use for CSRF validation: "cookie" or "hmac".  If unspecified, "hmac" is used if private_key is set to a value, otherwise "cookie" is used.
	// PrivateKey              string     `yaml:"private_key"             json:"private_key"`             // Provide a base64-encoded private key for use with the HMAC method of token validation
}

func (self *CSRF) GetHeaderName() string {
	if self.HeaderName != `` {
		return self.HeaderName
	} else {
		return DefaultCsrfHeaderName
	}
}

func (self *CSRF) GetFormFieldName() string {
	if self.FormFieldName != `` {
		return self.FormFieldName
	} else {
		return DefaultCsrfFormFieldName
	}
}

func (self *CSRF) GetCookieName() string {
	if c := self.Cookie; c != nil && c.Name != `` {
		return c.Name
	} else {
		return DefaultCsrfCookieName
	}
}

func (self *CSRF) Handle(w http.ResponseWriter, req *http.Request) bool {
	if self.Enable {
		log.Debugf("[%s] middleware: check csrf", reqid(req))
		self.generateTokenForRequest(w, req, false)

		switch req.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
			break
		default:
			if !self.IsExempt(req) {
				// if we're validating the request, then we've "consumed" this token and
				// should force-regenerate a new one
				self.generateTokenForRequest(w, req, true)
				creq := req.Clone(req.Context())

				if req.Body != nil {
					if body, err := ioutil.ReadAll(req.Body); err == nil {
						req.Body.Close()
						creq.Body = ioutil.NopCloser(bytes.NewBuffer(body))
						req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
					} else if self.server != nil {
						self.server.respondError(w, req, err, http.StatusBadRequest)
					} else {
						http.Error(w, err.Error(), http.StatusBadRequest)
						return false
					}
				}

				// if the token is missing/invalid, stop here and return an error
				if !self.Verify(creq) {
					if self.server != nil {
						self.server.respondError(w, req, fmt.Errorf("CSRF validation failed"), http.StatusBadRequest)
					} else {
						http.Error(w, "CSRF validation failed", http.StatusBadRequest)
					}

					return false
				}
			} else {
				log.Infof("[%s] path %q exempted from CSRF protection", reqid(req), req.URL.Path)
			}
		}
	}

	return true
}

// Retrieve the user-submitted token that can be forged.
func (self *CSRF) getUserSubmittedToken(req *http.Request) ([]byte, bool) {
	// first try to get the token from the header
	if token := req.Header.Get(self.GetHeaderName()); token != `` {
		return b58decode(token), true
	}

	// then try getting it from a form field
	if token := req.PostFormValue(self.GetFormFieldName()); token != `` {
		return b58decode(token), true
	}

	// Finally, try a multipart value.
	if req.MultipartForm != nil {
		if values, ok := req.MultipartForm.Value[self.GetFormFieldName()]; ok && len(values) > 0 && values[0] != `` {
			return b58decode(values[0]), true
		}
	}

	return nil, false
}

// Retrieve the cookie token that is harder to forge.
func (self *CSRF) getCookieToken(req *http.Request) ([]byte, bool) {
	if cookie, err := req.Cookie(self.GetCookieName()); err == nil {
		if cookie.Value != `` {
			return b58decode(cookie.Value), true
		}
	}

	return nil, false
}

// Verifies that the token that came in via the CSRF cookie and the one that came in
// as part of the request headers/body are, in fact, the same.
func (self *CSRF) Verify(req *http.Request) bool {
	if cookieToken, ok := self.getCookieToken(req); ok {
		if userToken, ok := self.getUserSubmittedToken(req); ok {
			if subtle.ConstantTimeCompare(cookieToken, userToken) == 1 {
				return true
			}
		}
	}

	return false
}

func (self *CSRF) cookieFor(token string) *http.Cookie {
	cookie := new(http.Cookie)
	cookie.Name = self.GetCookieName()
	cookie.Value = token
	cookie.Path = `/`
	cookie.MaxAge = 31536000

	if c := self.Cookie; c != nil {
		if c.MaxAge != nil {
			cookie.MaxAge = *c.MaxAge
		}

		if c.Secure != nil {
			cookie.Secure = *c.Secure
		}

		if c.HttpOnly != nil {
			cookie.HttpOnly = *c.HttpOnly
		}

		if c.Path != `` {
			cookie.Path = c.Path
		}

		if c.Domain != `` {
			cookie.Domain = c.Domain
		}

		if c.SameSite != `` {
			cookie.SameSite = c.SameSite.SameSite()
		}
	}

	return cookie
}

func (self *CSRF) generateTokenForRequest(w http.ResponseWriter, req *http.Request, forceRegen bool) {
	var data []byte

	if cookieToken, ok := self.getCookieToken(req); ok && len(cookieToken) == CsrfTokenLength && !forceRegen {
		data = cookieToken
	} else {
		data = make([]byte, CsrfTokenLength)

		if _, err := io.ReadFull(rand.Reader, data); err != nil {
			panic(err)
		}
	}

	token := b58encode(data)

	// attach token to the current request context so other things involved in
	// generating the response can see it
	httputil.RequestSetValue(req, ContextCsrfToken, token)

	// set the cookie
	w.Header().Set(`Vary`, `Cookie`)
	w.Header().Set(self.GetHeaderName(), token)
	cookie := self.cookieFor(token)
	http.SetCookie(w, cookie)
}

func (self *CSRF) shouldPostprocessRequest(w http.ResponseWriter, req *http.Request) bool {
	mediaTypes := DefaultCsrfInjectMediaTypes
	resMediaType := w.Header().Get(`Content-Type`)

	if len(self.InjectableMediaTypes) > 0 {
		mediaTypes = self.InjectableMediaTypes
	}

	for _, ct := range mediaTypes {
		if mt, _, err := mime.ParseMediaType(resMediaType); err == nil {
			if strings.ToLower(ct) == strings.ToLower(mt) {
				return true
			}
		}
	}

	return false
}

func (self *CSRF) IsExempt(req *http.Request) bool {
	if req != nil {
		for _, pattern := range self.Except {
			if m, err := filepath.Match(pattern, req.URL.Path); err == nil && m {
				return true
			}
		}
	}

	return false
}

func (self *Server) middlewareCsrf(w http.ResponseWriter, req *http.Request) bool {
	// enforce CSRF protection (if configured)
	if csrf := self.CSRF; csrf != nil && csrf.Enable {
		if !csrf.registered {
			csrf.server = self

			// Okay, so...
			//
			// Cross-Site Request Forgery is an insane problem that we have in modern web browsers in which
			// an authenticated user that is totally allowed to make requests can be tricked (using trickery)
			// into making a valid HTTP request that they did not intend to make.  99% of the time this is done
			// by somehow getting them to execute JavaScript in their browser that does Nasty Stuff™.
			//
			// How do to protect a user from this insane problem that we should have better solutions to by now?
			//
			// With each request, you include a one-time use token that is set in two places: a cookie and somewhere
			// in the content itself (e.g.: a hidden form field or an HTTP header).  If the token submitted from the
			// form doesn't match the value submitted in the cookie, you're a hacking hacker and its Bad News Time.
			//
			// "But updating a bunch of forms I may or may not control is annoying and difficult?"
			//
			// SURE IS!
			//
			// So, if you set this lunatic feature to "true", here's what Diecast will do for any content that is
			// CSRF protected (as defined in the csrf.except setting):
			//
			// 1. Is csrf.enable set to true?
			// 2. YES!  Attempt to parse the content as an HTML document.
			// 3. COOL! Select all elements from that document that match csrf.injectFormFieldSelector
			// 4. RAD!  Append the element described in csrf.formTokenTagFormat to those matching elements.
			// 5. NEAT! Serve *THAT* HTML instead.
			//
			// "Yeah we ran out of floorboards so we just painted the dirt. Pretty Clever!"
			//
			RegisterPostprocessor(`__diecast_csrf`, func(in string, req *http.Request) (string, error) {
				if req != nil {
					w := reqres(req)

					if csrf.shouldPostprocessRequest(w, req) {
						if csrf.InjectFormFields {
							if csrf.InjectFormFieldSelector == `` {
								csrf.InjectFormFieldSelector = DefaultCsrfInjectFormFieldSelector
							}

							if csrf.InjectFormFieldTemplate == `` {
								csrf.InjectFormFieldTemplate = DefaultCsrfInjectFieldFormat
							}

							log.Debugf("[%s] injecting form field", reqid(req))

							start := time.Now()
							defer reqtime(req, `csrf-inject`, time.Since(start))

							if doc, err := htmldoc(in); err == nil {
								doc.Find(csrf.InjectFormFieldSelector).Each(func(i int, form *goquery.Selection) {
									if form.Find(fmt.Sprintf("input[name=%q]", csrf.GetFormFieldName())).Length() == 0 {
										form.AppendHtml(
											fmt.Sprintf(
												csrf.InjectFormFieldTemplate,
												csrf.GetFormFieldName(),
												csrftoken(req),
											),
										)
									}
								})

								doc.End()

								if h, err := doc.Html(); err == nil {
									w.Header().Set(`Content-Length`, typeutil.String(len(h)))
									return h, nil
								} else {
									return ``, err
								}
							}
						}
					}
				}

				return in, nil
			})

			if self.BaseHeader == nil {
				self.BaseHeader = new(TemplateHeader)
			}

			self.BaseHeader.Postprocessors = append([]string{`__diecast_csrf`}, self.BaseHeader.Postprocessors...)
			csrf.registered = true
		}

		return csrf.Handle(w, req)
	} else {
		return true
	}
}
