package internal

import (
	"fmt"
	"io"
	"time"

	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/typeutil"
	"github.com/pkg/errors"
)

type DataSourceResponseFormat int

func (format DataSourceResponseFormat) MarshalYAML() (any, error) {
	switch format {
	case FormatRaw:
		return `raw`, nil
	case FormatArray:
		return `array`, nil
	default:
		return ``, nil
	}
}

func (format *DataSourceResponseFormat) UnmarshalYAML(unmarshal func(any) error) error {
	var incomingValue string

	if err := unmarshal(&incomingValue); err == nil {
		switch incomingValue {
		case `raw`:
			*format = FormatRaw
		case `array`:
			*format = FormatArray
		case ``, `map`:
			*format = FormatMap
		default:
			return fmt.Errorf("unknown format %q", incomingValue)
		}

		return nil
	} else {
		return err
	}
}

func (format DataSourceResponseFormat) Parse(data io.Reader) (any, error) {
	switch format {
	case FormatRaw:
		return data, nil
	case FormatArray:
		return []any{`NOT`, `IMPLEMENTED`}, nil
	case FormatMap:
		return maputil.M(data).MapNative(), nil
	default:
		return nil, fmt.Errorf("unknown format %q", format)
	}
}

const (
	FormatMap DataSourceResponseFormat = iota
	FormatArray
	FormatRaw
)

type DataSource struct {
	Content           any                      `yaml:"content,omitempty"`
	ID                string                   `yaml:"id"`
	Insecure          bool                     `yaml:"insecure"`
	Isolated          bool                     `yaml:"isolated"`
	Optional          bool                     `yaml:"optional"`
	RequestHeaders    map[string]any           `yaml:"headers"`
	RequestMethod     string                   `yaml:"method,omitempty"`
	RequestParameters map[string]any           `yaml:"params"`
	ResponseParser    DataSourceResponseFormat `yaml:"parser"`
	Timeout           time.Duration            `yaml:"timeout"`
	Transform         any                      `yaml:"transform,omitempty"`
	URL               string                   `yaml:"url"`
}

// Retrieve the current value as configured by this datasource, including all
// response data parsing and transformations.
func (self DataSource) Retrieve(ctx Contextable) (any, error) {
	if u := ctx.T(self.URL).String(); u != `` {
		if rc, err := RetrieveData(ctx, &RetrieveOptions{
			Headers:  StringMapEval(ctx, self.RequestHeaders),
			Insecure: self.Insecure,
			Isolated: self.Isolated,
			Method:   self.RequestMethod,
			Params:   MapEval(ctx, self.RequestParameters),
			Timeout:  self.Timeout,
			URL:      typeutil.String(MustEval(ctx, self.URL)),
		}); err == nil {
			defer rc.Close()

			if parsed, err := self.ResponseParser.Parse(rc); err == nil {
				return parsed, nil
			} else {
				return nil, errors.Wrapf(err, "datasource %q", self.ID)
			}
		} else if !self.Optional {
			return nil, errors.Wrapf(err, "datasource %q", self.ID)
		}
	}

	var v, err = ctx.Eval(self.Content)
	return v.Value, err
}

type DataSet []DataSource

func (self DataSet) Retrieve(ctx Contextable) (map[string]any, error) {
	for i, ds := range self {
		var target = ctx.T(ds.ID).String()

		if target == `` {
			return nil, fmt.Errorf("datasource %d: id must be set", i)
		}

		if v, err := ds.Retrieve(ctx); err == nil {
			ctx.SetValue(`data.`+target, v)
		} else if err.Error() == `skip` {
			continue
		} else {
			return nil, fmt.Errorf("datasource %q: %v", ds.ID, err)
		}
	}

	return ctx.Data(), nil
}
