package internal

import (
	"fmt"
	"time"

	"github.com/ghetzel/go-stockutil/executil"
	"github.com/ghetzel/go-stockutil/typeutil"
	"github.com/pkg/errors"
)

var DefaultResponseParser = executil.Env(`DIECAST_DS_DEFAULT_PARSER`, `json`)

type DataSource struct {
	Content           any            `yaml:"content,omitempty"`
	Disable           bool           `yaml:"disable,omitempty"`
	ID                string         `yaml:"id"`
	Insecure          bool           `yaml:"insecure"`
	Isolated          bool           `yaml:"isolated"`
	Optional          bool           `yaml:"optional"`
	RequestHeaders    map[string]any `yaml:"headers"`
	RequestMethod     string         `yaml:"method,omitempty"`
	RequestParameters map[string]any `yaml:"params"`
	ResponseParser    string         `yaml:"parser"`
	Timeout           time.Duration  `yaml:"timeout"`
	Transformations   DataTransforms `yaml:"transforms,omitempty"`
	URL               string         `yaml:"url"`
}

// Retrieve the current value as configured by this datasource, including all
// response data parsing and transformations.
func (self DataSource) Retrieve(ctx Contextable) (any, error) {
	var returnValue any

	if u := ctx.T(self.URL).String(); u != `` {
		var opts = &RetrieveOptions{
			Headers:  StringMapEval(ctx, self.RequestHeaders),
			Insecure: self.Insecure,
			Isolated: self.Isolated,
			Method:   self.RequestMethod,
			Params:   MapEval(ctx, self.RequestParameters),
			Timeout:  self.Timeout,
			URL:      typeutil.String(MustEval(ctx, self.URL)),
		}

		if rc, err := RetrieveData(ctx, opts); err == nil {
			defer rc.Close()

			ctx.Debugf("  ${green}\u2B82${reset} Datasource %q: %v", self.ID, opts.URL)

			if self.ResponseParser == `` {
				self.ResponseParser = DefaultResponseParser
			}

			if parser, ok := responseParsers[self.ResponseParser]; ok && parser != nil {
				if v, err := parser(rc); err == nil {
					returnValue = v
				} else {
					return nil, err
				}
			} else {
				return nil, errors.Wrapf(err, "undefined parser %q", self.ResponseParser)
			}
		} else if !self.Optional {
			ctx.Debugf("  ${red+b}\u2B82 Datasource %q: %v${reset}", self.ID, err)
			return nil, err
		} else {
			ctx.Debugf("  ${blue}\u2B82 Datasource %q [optional]: %v${reset}", self.ID, err)
		}
	}

	if returnValue == nil {
		if v, err := ctx.Eval(self.Content); err == nil {
			returnValue = v
		} else {
			return nil, err
		}
	}

	if len(self.Transformations) > 0 {
		if transformed, err := self.Transformations.Transform(returnValue); err == nil {
			returnValue = transformed
		} else {
			return nil, err
		}
	}

	return returnValue, nil
}

type DataSet []DataSource

func (self DataSet) Retrieve(ctx Contextable) (map[string]any, error) {
	if len(self) > 0 {
		for i, ds := range self {
			if ds.Disable {
				continue
			}

			var target = ctx.T(ds.ID).String()

			if target == `` {
				return nil, fmt.Errorf("datasource %d: id must be set", i)
			}

			if v, err := ds.Retrieve(ctx); err == nil {
				ctx.SetValue(`data.`+target, typeutil.ResolveValue(v))
			} else if err.Error() == `skip` {
				continue
			} else {
				return nil, fmt.Errorf("datasource %q: %v", ds.ID, err)
			}
		}
	}

	return ctx.Data(), nil
}
