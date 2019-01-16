package diecast

import (
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/pathutil"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/typeutil"
	htpasswd "github.com/tg123/go-htpasswd"
)

type BasicAuthenticator struct {
	htpasswd []*htpasswd.HtpasswdFile
	realm    string
}

func NewBasicAuthenticator(options map[string]interface{}) (*BasicAuthenticator, error) {
	auth := &BasicAuthenticator{
		realm: fmt.Sprintf("diecast/%v", ApplicationVersion),
	}

	if realm, ok := options[`realm`]; ok {
		auth.realm = typeutil.V(realm).String()
	}

	htpasswds := sliceutil.Stringify(sliceutil.Compact(options[`htpasswd`]))

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

func (self *BasicAuthenticator) Callback(w http.ResponseWriter, req *http.Request) error {
	return nil
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
