package internal

import (
	"fmt"
	"time"

	"github.com/ghetzel/go-stockutil/maputil"
)

type DataSource struct {
	Content           any            `yaml:"content,omitempty"`
	ID                string         `yaml:"id"`
	Insecure          bool           `yaml:"insecure"`
	Isolated          bool           `yaml:"isolated"`
	RequestHeaders    map[string]any `yaml:"headers"`
	RequestMethod     string         `yaml:"method,omitempty"`
	RequestParameters map[string]any `yaml:"params"`
	Timeout           time.Duration  `yaml:"timeout"`
	Transform         any            `yaml:"transform,omitempty"`
	URL               string         `yaml:"url"`
}

func (self DataSource) Retrieve(ctx Contextable) (any, error) {
	if u := ctx.T(self.URL).String(); u != `` {
		return RetrieveData(ctx, &RetrieveOptions{
			Fallback: self.Content,
			Headers:  maputil.Stringify(self.RequestHeaders),
			Insecure: self.Insecure,
			Isolated: self.Isolated,
			Method:   self.RequestMethod,
			Params:   self.RequestParameters,
			Timeout:  self.Timeout,
			URL:      self.URL,
		})
	} else if self.Content != nil {
		return self.Content, nil
	} else {
		return nil, fmt.Errorf(`skip`)
	}
}

type DataSet []DataSource

func (self DataSet) Retrieve(ctx Contextable) (map[string]any, error) {
	for i, ds := range self {
		var target = ctx.T(ds.ID).String()

		if target == `` {
			return nil, fmt.Errorf("datasource %d: id must be set", i)
		}

		if v, err := ds.Retrieve(ctx); err == nil {
			ctx.SetValue(target, v)
		} else if err.Error() == `skip` {
			continue
		} else {
			return nil, fmt.Errorf("datasource %q: %v", ds.ID, err)
		}
	}

	return ctx.Data(), nil
}
