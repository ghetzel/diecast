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
	htpasswd "github.com/tg123/go-htpasswd"
)

type BasicAuthenticator struct {
	config   *AuthenticatorConfig
	htpasswd []*htpasswd.HtpasswdFile
	realm    string
}

func NewBasicAuthenticator(config *AuthenticatorConfig) (*BasicAuthenticator, error) {
	auth := &BasicAuthenticator{
		config: config,
		realm:  config.O(`realm`, fmt.Sprintf("diecast/%v", ApplicationVersion)).String(),
	}

	htpasswds := sliceutil.Stringify(sliceutil.Compact(config.O(`htpasswd`).Value))

	if len(htpasswds) == 0 {
		return nil, fmt.Errorf("Must specify at least one user database via the 'htpasswd' option")
	} else {
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

	return auth, nil
}

func (self *BasicAuthenticator) Name() string {
	if self.config != nil && self.config.Name != `` {
		return self.config.Name
	} else {
		return `BasicAuthenticator`
	}
}

func (self *BasicAuthenticator) AddPasswdFile(filename string) error {
	if htp, err := htpasswd.New(filename, htpasswd.DefaultSystems, func(err error) {
		log.Warningf("BasicAuthenticator: %v", err)
	}); err == nil {
		self.htpasswd = append(self.htpasswd, htp)
		return nil
	} else {
		return err
	}
}

func (self *BasicAuthenticator) IsCallback(_ *url.URL) bool {
	return false
}

func (self *BasicAuthenticator) Callback(w http.ResponseWriter, req *http.Request) {

}

func (self *BasicAuthenticator) Authenticate(w http.ResponseWriter, req *http.Request) bool {
	if _, uppair := stringutil.SplitPair(req.Header.Get("Authorization"), ` `); uppair != `` {
		if decoded, err := base64.StdEncoding.DecodeString(uppair); err == nil {
			username, password := stringutil.SplitPair(string(decoded), `:`)

			for _, htp := range self.htpasswd {
				if htp.Match(username, password) {
					return true
				}
			}
		} else {
			log.Warningf("malformed authorization header")
		}

		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`Authorization Failed`))
		return false
	} else {
		wwwauth := `Basic`

		if self.realm != `` {
			wwwauth += ` realm=` + self.realm
		}

		w.Header().Set(`WWW-Authenticate`, wwwauth)
		w.WriteHeader(http.StatusUnauthorized)
		return false
	}
}
