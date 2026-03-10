package diecast

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/ghetzel/go-stockutil/typeutil"
)

type BasicAuthValidator struct{}

func (self *BasicAuthValidator) Validate(ctx *Context, cfg *ValidatorConfig) (verr error) {
	if server := ctx.Server(); server != nil {
		if req := ctx.Request(); req != nil {
			if u, p, ok := req.BasicAuth(); ok {
				if err := server.BasicAuthenticate(u, p); err == nil {
					return
				} else {
					verr = err
				}
			} else {
				verr = errors.New("authentication not provided")
			}
		} else {
			verr = errors.New("request context not available")
		}
	} else {
		verr = errors.New("server instance not available")
	}

	if cfg.Optional {
		verr = nil
	} else {
		var props []string

		if len(cfg.Options) == 0 {
			cfg.Options = make(map[string]any)
		}

		if v := cfg.Options[`realm`]; v == nil {
			cfg.Options[`realm`] = `restricted`
		}

		for k, v := range cfg.Options {
			if typeutil.IsKindOfString(v) {
				v = fmt.Sprintf("%q", v)
			}

			props = append(props, k+`=`+typeutil.String(v))
		}

		sort.Slice(props, func(i, j int) bool {
			return props[i] < props[j]
		})

		ctx.SetStatusCode(http.StatusUnauthorized)
		ctx.Header().Set(`WWW-Authenticate`, `Basic `+strings.Join(props, `, `))

		if verr == nil {
			verr = errors.New("authentication required")
		}
	}

	return
}
