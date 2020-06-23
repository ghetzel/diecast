package diecast

import (
	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/sliceutil"
)

type Redirect struct {
	URL  string `yaml:"url"  json:"url"`
	Code int    `yaml:"code" json:"code"`
}

type SwitchCase struct {
	CheckType string `yaml:"type,omitempty"      json:"type,omitempty"`      // The type of test to perform (one of: "expression", "querystring:<name>", "header:<name>")
	Condition string `yaml:"condition,omitempty" json:"condition,omitempty"` // A type-specific condition value (e.g.: an expression or querystring value)
	UsePath   string `yaml:"use,omitempty"       json:"use,omitempty"`       // The template to load if the condition matches
}

type TemplateHeader struct {
	Page              map[string]interface{} `yaml:"page,omitempty"            json:"page,omitempty"`            // An object that is accessible to this template (and all inheriting templates) under the `$.page` variable.
	Bindings          []Binding              `yaml:"bindings,omitempty"        json:"bindings,omitempty"`        // An array of remote URLs to to be retrieved (in order) and added to the `$.bindings` object.
	Defaults          map[string]string      `yaml:"defaults,omitempty"        json:"defaults,omitempty"`        // An object containing default query string values that can be accessed via the `qs` function.
	DefaultHeaders    map[string]string      `yaml:"default_headers,omitempty" json:"default_headers,omitempty"` // An object containing default HTTP request header values that can be accessed via the `$.request.headers` variable.
	Redirect          *Redirect              `yaml:"redirect,omitempty"        json:"redirect,omitempty"`        // Specifies an HTTP redirect should be performed when this page is accessed.
	Switch            []*SwitchCase          `yaml:"switch,omitempty"          json:"switch,omitempty"`          // Specify which template file to load in lieu of this one depending on which condition evaluates true first.
	Layout            string                 `yaml:"layout,omitempty"          json:"layout,omitempty"`          // The name of the layout (in the _layouts folder) to apply to this template.
	Includes          map[string]string      `yaml:"includes,omitempty"        json:"includes,omitempty"`        // An object specifying a custom name and path to other templates to include when evaluating this one.
	Headers           map[string]interface{} `yaml:"headers,omitempty"         json:"headers,omitempty"`         // A map of HTTP Headers to include in the request
	UrlParams         []KV                   `yaml:"url_params,omitempty"      json:"url_params,omitempty"`      // A map of query string parameters to include in the request
	FlagDefs          map[string]interface{} `yaml:"flags,omitempty"           json:"flags,omitempty"`           // An object containing names and expressions to add to a `$.flags` variable.
	Postprocessors    []string               `yaml:"postprocessors,omitempty"  json:"postprocessors,omitempty"`  // An array of built-in postprocessors to apply to the output before being returned to the user.
	Renderer          string                 `yaml:"renderer,omitempty"        json:"renderer,omitempty"`        // The built-in renderer to use when generating the page.
	Translations      map[string]interface{} `yaml:"translations,omitempty"    json:"translations,omitempty"`    // Stores translations for use with the i18n and l10n functions.  Keys values represent the
	Locale            string                 `yaml:"locale"                    json:"locale"`                    // Stores the locale used for this page.  If Locale is set on multiple levels of rendering, the last evaluated value is used.
	QueryJoiner       string                 `yaml:"query_joiner,omitempty"    json:"query_joiner,omitempty"`    // Override the string used to join multiple values of the same query string parameter.
	HeaderJoiner      string                 `yaml:"header_joiner,omitempty"   json:"header_joiner,omitempty"`   // Override the string used to join multiple values of the same HTTP header.
	StatusCode        int                    `yaml:"code,omitempty"            json:"code,omitempty"`            // Override the HTTP response status code of this page
	lines             int
	additionalHeaders map[string]interface{}
}

func (self *TemplateHeader) Merge(other *TemplateHeader) (*TemplateHeader, error) {
	if other == nil {
		return self, nil
	}

	var newHeader = &TemplateHeader{
		Bindings:       append(self.Bindings, other.Bindings...),             // ours first, then other's
		Layout:         sliceutil.OrString(other.Layout, self.Layout),        // prefer other, fallback to ours
		Renderer:       sliceutil.OrString(other.Renderer, self.Renderer),    // prefer other, fallback to ours
		Postprocessors: append(self.Postprocessors, other.Postprocessors...), // ours first, then other's
		Switch:         append(self.Switch, other.Switch...),                 // ours first, then other's
	}

	// locale: latest non-empty locale wins
	if other.Locale != `` {
		newHeader.Locale = other.Locale
	} else {
		newHeader.Locale = self.Locale
	}

	// joiners
	if other.QueryJoiner != `` {
		newHeader.QueryJoiner = other.QueryJoiner
	}

	if other.HeaderJoiner != `` {
		newHeader.HeaderJoiner = other.HeaderJoiner
	}

	// locale: latest non-empty locale wins
	if other.Locale != `` {
		newHeader.Locale = other.Locale
	} else {
		newHeader.Locale = self.Locale
	}

	// status code: prefer non-zero
	if other.StatusCode != 0 {
		newHeader.StatusCode = other.StatusCode
	} else {
		newHeader.StatusCode = self.StatusCode
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

	if v, err := maputil.Merge(self.Translations, other.Translations); err == nil {
		newHeader.Translations = v
	} else {
		return nil, err
	}

	return newHeader, nil
}
