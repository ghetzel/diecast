package diecast

import (
	"fmt"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/typeutil"
	"github.com/justinas/nosurf"
)

const DefaultCsrfInjectFormFieldSelector = `form[method="post"], form[method="POST"], form[method="Post"]` // if you need more case permutations than this, you may override this default
const DefaultCsrfInjectFieldFormat = `<input type="hidden" name="csrf_token" value="%s">`

var DefaultCsrfInjectMediaTypes = []string{
	`text/html`,
}

type CookieSameSite string

const (
	SameSiteDefault CookieSameSite = ``
	SameSiteLax                    = `lax`
	SameSiteStrict                 = `strict`
	SameSiteNone                   = `none`
)

func (self CookieSameSite) SameSite() http.SameSite {
	switch self {
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

type Cookie struct {
	Name     string         `yaml:"name,omitempty"     json:"name,omitempty"`
	Path     string         `yaml:"path,omitempty"     json:"path,omitempty"`
	Domain   string         `yaml:"domain,omitempty"   json:"domain,omitempty"`
	MaxAge   int            `yaml:"maxAge,omitempty"   json:"maxAge,omitempty"`
	Secure   bool           `yaml:"secure,omitempty"   json:"secure,omitempty"`
	HttpOnly bool           `yaml:"httpOnly,omitempty" json:"httpOnly,omitempty"`
	SameSite CookieSameSite `yaml:"sameSite,omitempty" json:"sameSite,omitempty"`
}

type CSRF struct {
	Enable                  bool     `yaml:"enable"                  json:"enable"`                  // Whether to enable stateless CSRF protection
	Except                  []string `yaml:"except"                  json:"except"`                  // A list of paths and path globs that should not be covered by CSRF protection
	Cookie                  *Cookie  `yaml:"cookie"                  json:"cookie"`                  // Specify default fields for the CSRF cookie that is set
	InjectFormFields        bool     `yaml:"injectFormFields"        json:"injectFormFields"`        // If true, a postprocessor will be added that injects a hidden <input> field into all <form> elements returned from Diecast
	InjectFormFieldSelector string   `yaml:"injectFormFieldSelector" json:"injectFormFieldSelector"` // A CSS selector used to locate <form> tags that need the CSRF <input> field injected.
	FormTokenTagFormat      string   `yaml:"formTokenTagFormat"      json:"formTokenTagFormat"`      // Specify the format string that will be used to replace </form> tags with the injected field.
	InjectableMediaTypes    []string `yaml:"injectableMediaTypes"    json:"injectableMediaTypes"`    // Specify a list of Media Types (e.g.: MIME or Content-Types) that will have injection attempted on them (if enabled)
}

func (self *CSRF) IsExempt(req *http.Request) bool {
	if req != nil {
		for _, pattern := range self.Except {
			if m, err := filepath.Match(pattern, req.URL.Path); err == nil && m {
				log.Infof("[%s] path %q exempted from CSRF protection", reqid(req), req.URL.Path)
				return true
			}
		}
	}

	return false
}

func (self *Server) applyCsrfIntercept() {
	var hnd http.Handler = self.router

	// enforce CSRF protection (if configured)
	if csrf := self.CSRF; csrf != nil {
		if csrf.Enable {
			csrfhnd := nosurf.New(self.router)
			csrfhnd.ExemptFunc(csrf.IsExempt)

			if c := csrf.Cookie; c != nil {
				csrfhnd.SetBaseCookie(http.Cookie{
					Name:     c.Name,
					Path:     c.Path,
					Domain:   c.Domain,
					MaxAge:   c.MaxAge,
					Secure:   c.Secure,
					HttpOnly: c.HttpOnly,
					SameSite: c.SameSite.SameSite(),
				})
			}

			csrfhnd.SetFailureHandler(
				constantErrHandler(self, fmt.Errorf("CSRF verification failed"), http.StatusForbidden),
			)

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
			if csrf.InjectFormFields {
				if csrf.InjectFormFieldSelector == `` {
					csrf.InjectFormFieldSelector = DefaultCsrfInjectFormFieldSelector
				}

				if csrf.FormTokenTagFormat == `` {
					csrf.FormTokenTagFormat = DefaultCsrfInjectFieldFormat
				}

				RegisterPostprocessor(`__diecast_csrf`, func(in string, req *http.Request) (string, error) {
					if req != nil {
						w := reqres(req)
						mediaTypes := DefaultCsrfInjectMediaTypes
						resMediaType := w.Header().Get(`Content-Type`)

						if len(csrf.InjectableMediaTypes) > 0 {
							mediaTypes = csrf.InjectableMediaTypes
						}

						var proceedWithCoolIdeas bool

						for _, ct := range mediaTypes {
							if mt, _, err := mime.ParseMediaType(resMediaType); err == nil {
								if strings.ToLower(ct) == strings.ToLower(mt) {
									proceedWithCoolIdeas = true
									break
								}
							}
						}

						if proceedWithCoolIdeas {
							log.Debugf("[%s] injecting csrf_token field (matched media type %s)", reqid(req), resMediaType)

							start := time.Now()
							defer reqtime(req, `csrf-inject`, time.Since(start))

							if doc, err := htmldoc(in); err == nil {
								doc.Find(csrf.InjectFormFieldSelector).AppendHtml(
									fmt.Sprintf(csrf.FormTokenTagFormat, nosurf.Token(req)),
								)

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

					return in, nil
				})

				if self.BaseHeader == nil {
					self.BaseHeader = new(TemplateHeader)
				}

				self.BaseHeader.Postprocessors = append([]string{`__diecast_csrf`}, self.BaseHeader.Postprocessors...)
			}

			hnd = csrfhnd
		}
	}

	self.handler.UseHandler(hnd)
}
