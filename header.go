package diecast

import (
	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/sliceutil"
)

type Redirect struct {
	URL  string `json:"url"`
	Code int    `json:"code"`
}

type SwitchCase struct {
	Condition string `json:"condition,omitempty"`
	UsePath   string `json:"use,omitempty"`
}

type TemplateHeader struct {
	Page           map[string]interface{} `json:"page,omitempty"`
	Bindings       []Binding              `json:"bindings,omitempty"`
	Defaults       map[string]string      `json:"defaults,omitempty"`
	DefaultHeaders map[string]string      `json:"default_headers,omitempty"`
	Redirect       *Redirect              `json:"redirect,omitempty"`
	Switch         []*SwitchCase          `json:"switch,omitempty"`
	Layout         string                 `json:"layout,omitempty"`
	Includes       map[string]string      `json:"includes,omitempty"`
	Headers        map[string]interface{} `json:"headers,omitempty"`
	UrlParams      map[string]interface{} `json:"params,omitempty"`
	FlagDefs       map[string]interface{} `json:"flags,omitempty"`
	Postprocessors []string               `json:"postprocessors,omitempty"`
	Renderer       string                 `json:"renderer,omitempty"`
	lines          int
}

func (self *TemplateHeader) Merge(other *TemplateHeader) (*TemplateHeader, error) {
	if other == nil {
		return self, nil
	}

	newHeader := &TemplateHeader{
		Bindings:       append(self.Bindings, other.Bindings...),             // ours first, then other's
		Layout:         sliceutil.OrString(other.Layout, self.Layout),        // prefer other, fallback to ours
		Renderer:       sliceutil.OrString(other.Renderer, self.Renderer),    // prefer other, fallback to ours
		Postprocessors: append(self.Postprocessors, other.Postprocessors...), // ours first, then other's
		Switch:         append(self.Switch, other.Switch...),                 // ours first, then other's
	}

	// Redirect: prefer other, fallback to ours
	if redir, ok := sliceutil.Or(other.Redirect, self.Redirect).(*Redirect); ok {
		newHeader.Redirect = redir
	}

	// maps: merge other's over top of ours

	if v, err := maputil.Merge(self.Page, other.Page); err == nil {
		newHeader.Page = v
	} else {
		return nil, err
	}

	if v, err := maputil.Merge(self.FlagDefs, other.FlagDefs); err == nil {
		newHeader.FlagDefs = v
	} else {
		return nil, err
	}

	if v, err := maputil.Merge(self.Defaults, other.Defaults); err == nil {
		newHeader.Defaults = maputil.Stringify(v)
	} else {
		return nil, err
	}

	if v, err := maputil.Merge(self.DefaultHeaders, other.DefaultHeaders); err == nil {
		newHeader.DefaultHeaders = maputil.Stringify(v)
	} else {
		return nil, err
	}

	if v, err := maputil.Merge(self.Headers, other.Headers); err == nil {
		newHeader.Headers = v
	} else {
		return nil, err
	}

	if v, err := maputil.Merge(self.Includes, other.Includes); err == nil {
		newHeader.Includes = maputil.Stringify(v)
	} else {
		return nil, err
	}

	return newHeader, nil
}
