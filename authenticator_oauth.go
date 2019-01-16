package diecast

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/ghetzel/go-stockutil/httputil"
	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/typeutil"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/amazon"
	"golang.org/x/oauth2/facebook"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/gitlab"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/microsoft"
	"golang.org/x/oauth2/slack"
	"golang.org/x/oauth2/spotify"
)

var DefaultOauth2SessionCookieName = `DCO2SESSION`
var oauthSessions sync.Map

type oauthSession struct {
	State      string
	Code       string
	Properties map[string]interface{}
	Token      *oauth2.Token
	Scheme     string
	Domain     string
}

type OauthAuthenticator struct {
	oauth2config    *oauth2.Config
	cookieName      string
	sessionDuration time.Duration
}

func NewOauthAuthenticator(options map[string]interface{}) (*OauthAuthenticator, error) {
	auth := &OauthAuthenticator{
		cookieName: DefaultOauth2SessionCookieName,
	}

	if cn, ok := options[`cookie_name`]; ok && cn != nil {
		auth.cookieName = typeutil.String(cn)
	}

	if lt, ok := options[`lifetime`]; ok && lt != nil {
		if duration, err := time.ParseDuration(typeutil.String(lt)); err == nil {
			auth.sessionDuration = duration
		} else {
			return nil, err
		}
	}

	if cid, ok := options[`client_id`]; ok {
		if csec, ok := options[`secret`]; ok {
			if redirect, ok := options[`redirect`]; ok {
				scopes := typeutil.Strings(options[`scopes`])

				auth.oauth2config = &oauth2.Config{
					ClientID:     typeutil.String(cid),
					ClientSecret: typeutil.String(csec),
					RedirectURL:  typeutil.String(redirect),
					Scopes:       scopes,
				}

				switch endpoint := typeutil.String(options[`provider`]); endpoint {
				case `amazon`:
					auth.oauth2config.Endpoint = amazon.Endpoint
				case `facebook`:
					auth.oauth2config.Endpoint = facebook.Endpoint
				case `github`:
					auth.oauth2config.Endpoint = github.Endpoint
				case `gitlab`:
					auth.oauth2config.Endpoint = gitlab.Endpoint
				case `microsoft-live`:
					auth.oauth2config.Endpoint = microsoft.LiveConnectEndpoint
				case `slack`:
					auth.oauth2config.Endpoint = slack.Endpoint
				case `spotify`:
					auth.oauth2config.Endpoint = spotify.Endpoint
				case `google`:
					auth.oauth2config.Endpoint = google.Endpoint
				default:
					return nil, fmt.Errorf("Unrecognized OAuth2 endpoint %q", endpoint)
				}
			} else {
				return nil, fmt.Errorf("The 'redirect' option is required for OauthAuthenticator")
			}
		} else {
			return nil, fmt.Errorf("The 'secret' option is required for OauthAuthenticator")
		}
	} else {
		return nil, fmt.Errorf("The 'client_id' option is required for OauthAuthenticator")
	}

	return auth, nil
}

// OAuth2: Leg 2: receive callback from consent page, validate session, and set session cookie
func (self *OauthAuthenticator) Callback(w http.ResponseWriter, req *http.Request) error {
	sid := httputil.Q(req, `state`)
	code := httputil.Q(req, `code`)

	if sessionI, ok := oauthSessions.Load(sid); ok {
		if session, ok := sessionI.(*oauthSession); ok {
			if session.State == sid {
				if token, err := self.oauth2config.Exchange(oauth2.NoContext, code); err == nil {
					session.Code = code
					session.Token = token

					// give the client their session ID
					cookie := &http.Cookie{
						Name:     self.cookieName,
						Value:    session.State,
						Path:     `/`,
						Domain:   session.Domain,
						Secure:   (session.Scheme == `https`),
						HttpOnly: true,
						SameSite: http.SameSiteStrictMode,
					}

					if self.sessionDuration > 0 {
						cookie.Expires = time.Now().Add(self.sessionDuration)
					}

					http.SetCookie(w, cookie)

					return nil
				} else {
					return err
				}
			} else {
				return fmt.Errorf("Invalid OAuth2 session returned for callback")
			}
		} else {
			return fmt.Errorf("Invalid OAuth2 session object")
		}
	} else {
		return fmt.Errorf("OAuth2 session does not exist exists")
	}
}

func (self *OauthAuthenticator) Authenticate(w http.ResponseWriter, req *http.Request) bool {
	if cookie, err := req.Cookie(self.cookieName); err == nil {
		if sessionI, ok := oauthSessions.Load(cookie.Value); ok {
			if session, ok := sessionI.(*oauthSession); ok {
				if session.State == cookie.Value {
					if session.Token != nil {
						return true
					}
				}
			}
		}
	} else if err == http.ErrNoCookie {
		// OAuth2: Leg 1: generate session ID and redirect to auth page
		sid := stringutil.UUID().Base58()

		// store the session pre-authenticated stub
		oauthSessions.Store(sid, &oauthSession{
			State:      sid,
			Properties: make(map[string]interface{}),
			Domain:     req.URL.Host,
			Scheme:     req.URL.Scheme,
		})

		// ...then redirect them to the auth page
		http.Redirect(w, req, self.oauth2config.AuthCodeURL(sid), http.StatusTemporaryRedirect)
		return true
	}

	return false
}
