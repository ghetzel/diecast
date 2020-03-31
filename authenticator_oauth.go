package diecast

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/ghetzel/go-stockutil/httputil"
	"github.com/ghetzel/go-stockutil/stringutil"
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
	Path       string
}

type OauthAuthenticator struct {
	config          *AuthenticatorConfig
	oauth2config    *oauth2.Config
	cookieName      string
	sessionDuration time.Duration
}

func NewOauthAuthenticator(config *AuthenticatorConfig) (*OauthAuthenticator, error) {
	var auth = &OauthAuthenticator{
		config:          config,
		cookieName:      config.O(`cookie_name`, DefaultOauth2SessionCookieName).String(),
		sessionDuration: config.O(`lifetime`).Duration(),
		oauth2config: &oauth2.Config{
			ClientID:     config.O(`client_id`).String(),
			ClientSecret: config.O(`secret`).String(),
			RedirectURL:  config.CallbackPath,
			Scopes:       config.O(`scopes`).Strings(),
		},
	}

	if auth.oauth2config.ClientID == `` {
		return nil, fmt.Errorf("The 'client_id' option is required for OauthAuthenticator")
	}

	if auth.oauth2config.ClientSecret == `` {
		return nil, fmt.Errorf("The 'secret' option is required for OauthAuthenticator")
	}

	if auth.oauth2config.RedirectURL == `` {
		return nil, fmt.Errorf("The 'callback' option is required for OauthAuthenticator")
	}

	switch endpoint := config.O(`provider`).String(); endpoint {
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
		auth.oauth2config.Endpoint = oauth2.Endpoint{
			AuthURL:  config.O(`auth_url`).String(),
			TokenURL: config.O(`token_url`).String(),
		}

		if auth.oauth2config.Endpoint.AuthURL == `` || auth.oauth2config.Endpoint.TokenURL == `` {
			return nil, fmt.Errorf("Custom OAuth2 endpoint must specify the 'auth_url' and 'token_url' options.")
		}

		return nil, fmt.Errorf("Unrecognized OAuth2 endpoint %q", endpoint)
	}

	return auth, nil
}

func (self *OauthAuthenticator) Name() string {
	if self.config != nil && self.config.Name != `` {
		return self.config.Name
	} else {
		return `OauthAuthenticator`
	}
}

func (self *OauthAuthenticator) IsCallback(u *url.URL) bool {
	if self.config != nil {
		if cb, err := url.Parse(self.config.CallbackPath); err == nil {
			if strings.TrimSuffix(cb.Path, `/`) == strings.TrimSuffix(u.Path, `/`) {
				return true
			}
		}
	}

	return false
}

// OAuth2: Leg 2: receive callback from consent page, validate session, and set session cookie
func (self *OauthAuthenticator) Callback(w http.ResponseWriter, req *http.Request) {
	var sid = httputil.Q(req, `state`)
	var code = httputil.Q(req, `code`)

	if sessionI, ok := oauthSessions.Load(sid); ok {
		if session, ok := sessionI.(*oauthSession); ok {
			if session.State == sid {
				if token, err := self.oauth2config.Exchange(oauth2.NoContext, code); err == nil {
					session.Code = code
					session.Token = token

					// give the client their session ID
					var cookie = &http.Cookie{
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
					http.Redirect(w, req, session.Path, http.StatusTemporaryRedirect)
				} else {
					http.Error(w, err.Error(), http.StatusBadRequest)
				}
			} else {
				http.Error(w, "Invalid OAuth2 session returned for callback", http.StatusBadRequest)
			}
		} else {
			http.Error(w, "Invalid OAuth2 session object", http.StatusBadRequest)
		}
	} else {
		http.Error(w, "OAuth2 session does not exist exists", http.StatusBadRequest)
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
		var sid = stringutil.UUID().Base58()

		// store the session pre-authenticated stub
		oauthSessions.Store(sid, &oauthSession{
			State:      sid,
			Properties: make(map[string]interface{}),
			Domain:     req.URL.Host,
			Scheme:     req.URL.Scheme,
			Path:       req.URL.Path,
		})

		// ...then redirect them to the auth page
		http.Redirect(w, req, self.oauth2config.AuthCodeURL(sid), http.StatusTemporaryRedirect)
		return true
	}

	return false
}
