package diecast

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/pathutil"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/typeutil"
	htpasswd "github.com/tg123/go-htpasswd"
)

type BasicAuthenticator struct {
	config      *AuthenticatorConfig
	htpasswd    []*htpasswd.File
	credentials map[string]any
	realm       string
}

func NewBasicAuthenticator(config *AuthenticatorConfig) (*BasicAuthenticator, error) {
	var auth = &BasicAuthenticator{
		config: config,
		realm:  config.O(`realm`, fmt.Sprintf("diecast/%v", ApplicationVersion)).String(),
	}

	var htpasswds = sliceutil.Stringify(sliceutil.Compact(config.O(`htpasswd`).Value))

	if len(htpasswds) > 0 {
		for _, filename := range htpasswds {
			if ex, err := pathutil.ExpandUser(filename); err == nil {
				if err := auth.AddPasswdFile(ex); err != nil {
					return nil, err
				}
			} else {
				return nil, err
			}
		}
	}

	auth.credentials = config.O(`credentials`).MapNative()

	if len(auth.htpasswd) == 0 && len(auth.credentials) == 0 {
		return nil, fmt.Errorf("must specify at least one user database via the 'htpasswd' option")
	} else {
		return auth, nil
	}
}

func (auth *BasicAuthenticator) Name() string {
	if auth.config != nil && auth.config.Name != `` {
		return auth.config.Name
	} else {
		return `BasicAuthenticator`
	}
}

func (auth *BasicAuthenticator) AddPasswdFile(filename string) error {
	if htp, err := htpasswd.New(filename, htpasswd.DefaultSystems, func(err error) {
		log.Warningf("BasicAuthenticator: %v", err)
	}); err == nil {
		auth.htpasswd = append(auth.htpasswd, htp)
		return nil
	} else {
		return err
	}
}

func (auth *BasicAuthenticator) IsCallback(_ *url.URL) bool {
	return false
}

func (auth *BasicAuthenticator) Callback(w http.ResponseWriter, req *http.Request) {

}

func (auth *BasicAuthenticator) Authenticate(w http.ResponseWriter, req *http.Request) bool {
	if _, uppair := stringutil.SplitPair(req.Header.Get("Authorization"), ` `); uppair != `` {
		if decoded, err := base64.StdEncoding.DecodeString(uppair); err == nil {
			username, password := stringutil.SplitPair(string(decoded), `:`)

			// match against any loaded htpasswd files
			for _, htp := range auth.htpasswd {
				if htp.Match(username, password) {
					return true
				}
			}

			// match against statically-configured user:passhash pairs
			for authUser, passhash := range auth.credentials {
				if username == authUser {
					var ph = typeutil.String(passhash)

					if enc, err := htpasswd.AcceptBcrypt(ph); err == nil && enc != nil {
						return enc.MatchesPassword(password)
					} else if enc, err := htpasswd.AcceptMd5(ph); err == nil && enc != nil {
						return enc.MatchesPassword(password)
					} else if enc, err := htpasswd.AcceptSha(ph); err == nil && enc != nil {
						return enc.MatchesPassword(password)
					} else if enc, err := htpasswd.AcceptSsha(ph); err == nil && enc != nil {
						return enc.MatchesPassword(password)
					}
				}
			}
		} else {
			log.Warningf("malformed authorization header")
		}

		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`Authorization Failed`))
		return false
	} else {
		var wwwauth = `Basic`

		if auth.realm != `` {
			wwwauth += ` realm=` + auth.realm
		}

		w.Header().Set(`WWW-Authenticate`, wwwauth)
		w.WriteHeader(http.StatusUnauthorized)
		return false
	}
}
