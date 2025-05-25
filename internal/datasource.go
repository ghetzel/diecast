package internal

import "fmt"

type DataSource struct {
	ID        string `yaml:"id"`
	URL       string `yaml:"url"`
	Transform any    `yaml:"transform,omitempty"`
	Content   any    `yaml:"content,omitempty"`
}

func (self DataSource) Retrieve(ctx Contextable) (any, error) {
	if self.Content != nil {
		return self.Content, nil
	} else if u := ctx.T(self.URL).String(); u != `` {
		return RetrieveURL(ctx, u)
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
