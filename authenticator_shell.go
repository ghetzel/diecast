package diecast

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"

	"github.com/ghetzel/go-stockutil/httputil"
	"github.com/ghetzel/go-stockutil/log"
	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/typeutil"
	"github.com/gobwas/glob"
	shellwords "github.com/mattn/go-shellwords"
)

var DefaultShellSessionCookieName = `DCSESSION`

type ShellAuthenticator struct {
	config     *AuthenticatorConfig
	deauthPath glob.Glob
}

func NewShellAuthenticator(config *AuthenticatorConfig) (*ShellAuthenticator, error) {
	auth := &ShellAuthenticator{
		config: config,
	}

	if deauth := config.O(`deauth_path`).String(); deauth != `` {
		if g, err := glob.Compile(deauth); err == nil {
			auth.deauthPath = g
		} else {
			return nil, fmt.Errorf("deauth_path: %v", err)
		}
	}

	if typeutil.IsEmpty(config.O(`command`).Value) {
		return nil, fmt.Errorf("command: cannot be empty")
	}

	return auth, nil
}

func (self *ShellAuthenticator) IsCallback(_ *url.URL) bool {
	return false
}

func (self *ShellAuthenticator) Callback(w http.ResponseWriter, req *http.Request) {

}

func (self *ShellAuthenticator) Authenticate(w http.ResponseWriter, req *http.Request) bool {
	id := reqid(req)
	config := self.config

	disableCookies := config.O(`disable_cookies`).Bool()
	var action string
	var stdin io.Reader
	var body map[string]interface{}

	if req.ContentLength != 0 {
		if err := httputil.ParseRequest(req, &body); err != nil {
			log.Warningf("[%s] %T: parse error: %v", id, self, err)
			return false
		}
	}

	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)

	// retrieve the session token data. if it's not present or we've disabled cookies, authenticate. else, validate.
	if cookie, err := req.Cookie(config.O(`cookie_name`, DefaultShellSessionCookieName).String()); disableCookies || err == http.ErrNoCookie {
		action = `create`
		stdin = nil
	} else if self.deauthPath != nil && self.deauthPath.Match(req.URL.Path) {
		action = `remove`
		stdin = bytes.NewBufferString(cookie.Value)
	} else {
		action = `verify`
		stdin = bytes.NewBufferString(cookie.Value)
	}

	var cmd *exec.Cmd

	if v := config.O(`command`); typeutil.IsArray(v.Value) {
		args := v.Strings()
		cmd = exec.Command(args[0], args[1:]...)
	} else if cmdline := v.String(); cmdline != `` {
		if args, err := shellwords.Parse(cmdline); err == nil {
			cmd = exec.Command(args[0], args[1:]...)
		} else {
			log.Warningf("[%s] %T: invalid command: %v", id, self, err)
			return false
		}
	} else {
		log.Warningf("[%s] %T: empty command", id, self)
		return false
	}

	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	cmd.Env = append(cmd.Env, fmt.Sprintf("DIECAST_AUTH_ACTION=%s", action))
	cmd.Env = append(cmd.Env, fmt.Sprintf("DIECAST_AUTH_URL=%s", req.URL.String()))
	cmd.Env = append(cmd.Env, fmt.Sprintf("DIECAST_AUTH_PATH=%s", req.URL.Path))
	cmd.Env = append(cmd.Env, fmt.Sprintf("DIECAST_AUTH_METHOD=%s", req.Method))
	cmd.Env = append(cmd.Env, fmt.Sprintf("DIECAST_AUTH_REMOTE_ADDR=%s", req.RemoteAddr))

	if flat, err := maputil.CoalesceMap(body, `__`); err == nil {
		for k, v := range flat {
			cmd.Env = append(cmd.Env, fmt.Sprintf("DIECAST_AUTH_BODY_%s=%v", strings.ToUpper(stringutil.Underscore(k)), v))
		}
	} else {
		log.Warningf("[%s] %T: body parse error: %v", id, self, err)
		return false
	}

	for k, vv := range req.Header {
		if len(vv) == 1 {
			cmd.Env = append(cmd.Env, fmt.Sprintf("DIECAST_AUTH_HEADER_%s=%v", strings.ToUpper(stringutil.Underscore(k)), vv[0]))
		} else {
			cmd.Env = append(cmd.Env, fmt.Sprintf("DIECAST_AUTH_HEADER_%s=%v", strings.ToUpper(stringutil.Underscore(k)), strings.Join(vv, `,`)))
		}
	}

	for k, vv := range req.URL.Query() {
		if len(vv) == 1 {
			cmd.Env = append(cmd.Env, fmt.Sprintf("DIECAST_AUTH_QUERY_%s=%v", strings.ToUpper(stringutil.Underscore(k)), vv[0]))
		} else {
			cmd.Env = append(cmd.Env, fmt.Sprintf("DIECAST_AUTH_QUERY_%s=%v", strings.ToUpper(stringutil.Underscore(k)), strings.Join(vv, `,`)))
		}
	}

	// execute the auth command.  nil error means it exited successfully
	if err := cmd.Run(); err == nil {
		// print any error output that came out
		for _, line := range strings.Split(stderr.String(), "\n") {
			log.Warningf("[%s] %T: error: %s", id, self, line)
		}

		// prep the cookie
		cookie := &http.Cookie{
			Name:     config.O(`cookie_name`, DefaultShellSessionCookieName).String(),
			Path:     config.O(`cookie_path`).String(),
			Domain:   config.O(`cookie_domain`).String(),
			HttpOnly: config.O(`cookie_http_only`).Bool(),
			Secure:   true,
			SameSite: http.SameSiteDefaultMode,
		}

		switch action {
		case `create`:
			if out := strings.TrimSpace(stdout.String()); !disableCookies && out != `` {
				cookie.Value = out

				if expiry := config.O(`cookie_lifetime`).Duration(); expiry > 0 {
					cookie.Expires = time.Now().Add(expiry)
				}

				if !config.O(`cookie_secure`).IsNil() {
					cookie.Secure = config.O(`cookie_secure`).Bool()
				}

				switch config.O(`cookie_samesite`).String() {
				case `lax`:
					cookie.SameSite = http.SameSiteLaxMode
				case `strict`:
					cookie.SameSite = http.SameSiteStrictMode
				}

				http.SetCookie(w, cookie)
			}

			return true

		case `verify`:
			return false

		case `remove`:
			cookie.Value = ``
			cookie.MaxAge = -1

			http.SetCookie(w, cookie)
		}
	}

	return false
}
