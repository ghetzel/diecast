package internal

import (
	"fmt"
	"time"

	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/timeutil"
	"github.com/ghetzel/go-stockutil/typeutil"
	"github.com/pkg/errors"
)

func legacyFeatureError(noun string) error {
	return fmt.Errorf("the %q field has been removed in Diecast 2.x", noun)
}

func legacyBindingFeatureError(id string, noun string) error {
	return fmt.Errorf("binding %q: the %q field has been removed in Diecast 2.x", id, noun)
}

type LegacyTemplateHeader struct {
	Page           map[string]any    `yaml:"page,omitempty"            json:"page,omitempty"`            // An object that is accessible to this template (and all inheriting templates) under the `$.page` variable.
	Bindings       []LegacyBinding   `yaml:"bindings,omitempty"        json:"bindings,omitempty"`        // An array of remote URLs to to be retrieved (in order) and added to the `$.bindings` object.
	Defaults       map[string]string `yaml:"defaults,omitempty"        json:"defaults,omitempty"`        // An object containing default query string values that can be accessed via the `qs` function.
	DefaultHeaders map[string]string `yaml:"default_headers,omitempty" json:"default_headers,omitempty"` // An object containing default HTTP request header values that can be accessed via the `$.request.headers` variable.
	Layout         string            `yaml:"layout,omitempty"          json:"layout,omitempty"`          // The name of the layout (in the _layouts folder) to apply to this template.
	Includes       map[string]string `yaml:"includes,omitempty"        json:"includes,omitempty"`        // An object specifying a custom name and path to other templates to include when evaluating this one.
	Headers        map[string]any    `yaml:"headers,omitempty"         json:"headers,omitempty"`         // A map of HTTP Headers to include in the response
	FlagDefs       map[string]any    `yaml:"flags,omitempty"           json:"flags,omitempty"`           // An object containing names and expressions to add to a `$.flags` variable.
	Postprocessors []string          `yaml:"postprocessors,omitempty"  json:"postprocessors,omitempty"`  // An array of built-in postprocessors to apply to the output before being returned to the user.
	Renderer       string            `yaml:"renderer,omitempty"        json:"renderer,omitempty"`        // The built-in renderer to use when generating the page.
	Translations   map[string]any    `yaml:"translations,omitempty"    json:"translations,omitempty"`    // Stores translations for use with the i18n and l10n functions.  Keys values represent the
	Locale         string            `yaml:"locale"                    json:"locale"`                    // Stores the locale used for this page.  If Locale is set on multiple levels of rendering, the last evaluated value is used.
	QueryJoiner    string            `yaml:"query_joiner,omitempty"    json:"query_joiner,omitempty"`    // Override the string used to join multiple values of the same query string parameter.
	HeaderJoiner   string            `yaml:"header_joiner,omitempty"   json:"header_joiner,omitempty"`   // Override the string used to join multiple values of the same HTTP header.
	StatusCode     int               `yaml:"code,omitempty"            json:"code,omitempty"`            // Override the HTTP response status code of this page
	// Redirect          *Redirect         `yaml:"redirect,omitempty"        json:"redirect,omitempty"`        // Specifies an HTTP redirect should be performed when this page is accessed.
	// Switch            []*SwitchCase     `yaml:"switch,omitempty"          json:"switch,omitempty"`          // Specify which template file to load in lieu of this one depending on which condition evaluates true first.
	// UrlParams         []KV              `yaml:"url_params,omitempty"      json:"url_params,omitempty"`      // A map of query string parameters to include in the request
}

func (legacy *LegacyTemplateHeader) convertToV2Header(v2hdr *TemplateHeader) error {
	if len(legacy.Defaults) > 0 {
		return legacyFeatureError("defaults")
	} else if len(legacy.DefaultHeaders) > 0 {
		return legacyFeatureError("default_headers")
	} else if len(legacy.FlagDefs) > 0 {
		return legacyFeatureError("flags")
	} else if legacy.Renderer != `` {
		return legacyFeatureError("renderer")
	} else if len(legacy.Translations) > 0 {
		return legacyFeatureError("translations")
	} else if legacy.Locale != `` {
		return legacyFeatureError("locale")
	} else if legacy.QueryJoiner != `` {
		return legacyFeatureError("query_joiner")
	} else if legacy.HeaderJoiner != `` {
		return legacyFeatureError("header_joiner")
	} else if legacy.StatusCode > 0 {
		return legacyFeatureError("code")
	}

	for _, binding := range legacy.Bindings {
		if ds, err := binding.convertToV2DataSource(); err == nil {
			v2hdr.DataSources = append(v2hdr.DataSources, *ds)
		} else {
			return errors.Wrapf(err, "legacy binding %q", binding.Name)
		}
	}

	v2hdr.Layout = legacy.Layout
	v2hdr.ResponseHeaders = legacy.Headers
	v2hdr.Page = legacy.Page

	return nil
}

type LegacyBinding struct {
	BodyParams         map[string]any    `yaml:"body,omitempty"                 json:"body,omitempty"`                 // If the request receives an open-ended body, this will allow structured data to be passed in.
	DisableCache       bool              `yaml:"disable_cache,omitempty"        json:"disable_cache,omitempty"`        // Reserved for future use.
	Fallback           any               `yaml:"fallback,omitempty"             json:"fallback,omitempty"`             // The value to place in $.bindings if the request fails.
	Formatter          string            `yaml:"formatter,omitempty"            json:"formatter,omitempty"`            // How to serialize BodyParams into a string before the request is made.
	Headers            map[string]string `yaml:"headers,omitempty"              json:"headers,omitempty"`              // Additional headers to include in the request.
	Insecure           bool              `yaml:"insecure,omitempty"             json:"insecure,omitempty"`             // If the protocol supports an insecure request mode (e.g.: HTTPS), permit it in this case.
	Method             string            `yaml:"method,omitempty"               json:"method,omitempty"`               // The protocol-specific method to perform the request with.
	Name               string            `yaml:"name,omitempty"                 json:"name,omitempty"`                 // The name of the key in the $.bindings template variable.
	NoTemplate         bool              `yaml:"no_template,omitempty"          json:"no_template,omitempty"`          // Disable templating of variables in this binding.
	NotIfExpr          string            `yaml:"not_if,omitempty"               json:"not_if,omitempty"`               // Do not evaluate the binding if this expression yields a truthy value.
	OnlyIfExpr         string            `yaml:"only_if,omitempty"              json:"only_if,omitempty"`              // Only evaluate the binding if this expression yields a truthy value.
	Optional           bool              `yaml:"optional,omitempty"             json:"optional,omitempty"`             // Whether the request failing will cause a page-wide error or be ignored.
	ParamJoiner        string            `yaml:"param_joiner,omitempty"         json:"param_joiner,omitempty"`         // If a parameter is provided as an array, but must be a string in the request, how shall the array elements be joined.
	Params             map[string]any    `yaml:"params,omitempty"               json:"params,omitempty"`               // A set of additional parameters to include in the request (e.g.: HTTP query string parameters)
	Parser             string            `yaml:"parser,omitempty"               json:"parser,omitempty"`               // How to parse the response content from the request.
	ProtocolOptions    map[string]any    `yaml:"protocol,omitempty"             json:"protocol,omitempty"`             // An open-ended set of options that are available for protocol implementations to use.
	RawBody            string            `yaml:"rawbody,omitempty"              json:"rawbody,omitempty"`              // If the request receives an open-ended body, this will allow raw data to be passed in as-is.
	Repeat             string            `yaml:"repeat,omitempty"               json:"repeat,omitempty"`               // A templated value that yields an array.  The binding request will be performed once for each array element, wherein the Resource value is passed into a template that includes the $index and $item variables, which represent the repeat array item's position and value, respectively.
	Resource           string            `yaml:"resource,omitempty"             json:"resource,omitempty"`             // The URL that specifies the protocol and resource to retrieve.
	SkipInheritHeaders bool              `yaml:"skip_inherit_headers,omitempty" json:"skip_inherit_headers,omitempty"` // Do not passthrough the headers that were sent to the template from the client's browser, even if Passthrough mode is enabled.
	Timeout            any               `yaml:"timeout,omitempty"              json:"timeout,omitempty"`              // A duration specifying the timeout for the request.
	Transform          string            `yaml:"transform,omitempty"            json:"transform,omitempty"`            // Specifies a JSONPath expression that can be used to transform the response data received from the binding into the data that is provided to the template.
	TlsCertificate     string            `yaml:"tlscrt,omitempty"               json:"tlscrt,omitempty"`               // Provide the path to a TLS client certificate to present if the server requests one.
	TlsKey             string            `yaml:"tlskey,omitempty"               json:"tlskey,omitempty"`               // Provide the path to a TLS client certificate key to present if the server requests one.
	OnlyPaths          []string          `yaml:"only,omitempty"                 json:"only,omitempty"`                 // A list of request paths and glob patterns, ANY of which the binding will evaluate on.
	ExceptPaths        []string          `yaml:"except,omitempty"               json:"except,omitempty"`               // A list of request paths and glob patterns, ANY of which the binding will NOT evaluate on.
	Interval           string            `yaml:"interval,omitempty"             json:"interval,omitempty"`             // For Async Bindings, this specifies the interval on which data sources should be refreshed (if so desired).
	Restrict           any               `yaml:"restrict,omitempty"`
	// IfStatus           map[string]BindingErrorAction `yaml:"if_status,omitempty"            json:"if_status,omitempty"`            // Actions to take in response to specific numeric response status codes.
	// OnError            BindingErrorAction            `yaml:"on_error,omitempty"             json:"on_error,omitempty"`             // Actions to take if the request fails.
	// Paginate           *PaginatorConfig              `yaml:"paginate,omitempty"             json:"paginate,omitempty"`             // A specialized repeater configuration that automatically performs pagination on an upstream request, aggregating the results before returning them.
}

func (binding *LegacyBinding) convertToV2DataSource() (*DataSource, error) {
	var dataSource = new(DataSource)

	if binding.Formatter != `` {
		return nil, legacyBindingFeatureError(binding.Name, "formatter")
	} else if binding.Repeat != `` {
		return nil, legacyBindingFeatureError(binding.Name, "repeat")
	} else if binding.NoTemplate {
		return nil, legacyBindingFeatureError(binding.Name, "no_template")
	} else if binding.TlsCertificate != `` {
		return nil, legacyBindingFeatureError(binding.Name, "tlscrt")
	} else if binding.TlsKey != `` {
		return nil, legacyBindingFeatureError(binding.Name, "tlskey")
	} else if len(binding.OnlyPaths) > 0 {
		return nil, legacyBindingFeatureError(binding.Name, "only")
	} else if len(binding.ExceptPaths) > 0 {
		return nil, legacyBindingFeatureError(binding.Name, "except")
	} else if binding.Interval != `` {
		return nil, legacyBindingFeatureError(binding.Name, "interval")
	}

	dataSource.Content = binding.RawBody
	dataSource.ID = binding.Name
	dataSource.Insecure = binding.Insecure
	dataSource.Isolated = binding.SkipInheritHeaders
	dataSource.RequestHeaders = maputil.Autotype(binding.Headers)
	dataSource.RequestMethod = binding.Method
	dataSource.RequestParameters = binding.Params
	dataSource.Transform = binding.Transform
	dataSource.URL = binding.Resource

	// if expr := binding.NotIfExpr; expr != `` {
	// 	dataSource.AddCondition(MustNotBeTruthy, expr)
	// }

	// if expr := binding.OnlyIfExpr; expr != `` {
	// 	dataSource.AddCondition(MustByTruthy, expr)
	// }

	if lto := binding.Timeout; lto != nil {
		if dur, ok := lto.(time.Duration); ok {
			dataSource.Timeout = dur
		} else if dur, ok := lto.(*time.Duration); ok {
			dataSource.Timeout = *dur
		} else if t, err := timeutil.ParseDuration(
			typeutil.String(lto),
		); err == nil {
			dataSource.Timeout = t
		} else {
			return nil, err
		}
	}

	return dataSource, nil
}

// RawBody        string   `yaml:"rawbody,omitempty"              json:"rawbody,omitempty"`   // If the request receives an open-ended body, this will allow raw data to be passed in as-is.
// Repeat         string   `yaml:"repeat,omitempty"               json:"repeat,omitempty"`    // A templated value that yields an array.  The binding request will be performed once for each array element, wherein the Resource value is passed into a template that includes the $index and $item variables, which represent the repeat array item's position and value, respectively.
// Timeout        any      `yaml:"timeout,omitempty"              json:"timeout,omitempty"`   // A duration specifying the timeout for the request.
// TlsCertificate string   `yaml:"tlscrt,omitempty"               json:"tlscrt,omitempty"`    // Provide the path to a TLS client certificate to present if the server requests one.
// TlsKey         string   `yaml:"tlskey,omitempty"               json:"tlskey,omitempty"`    // Provide the path to a TLS client certificate key to present if the server requests one.
// OnlyPaths      []string `yaml:"only,omitempty"                 json:"only,omitempty"`      // A list of request paths and glob patterns, ANY of which the binding will evaluate on.
// ExceptPaths    []string `yaml:"except,omitempty"               json:"except,omitempty"`    // A list of request paths and glob patterns, ANY of which the binding will NOT evaluate on.

// ProtocolOptions map[string]any `yaml:"protocol,omitempty"             json:"protocol,omitempty"` // An open-ended set of options that are available for protocol implementations to use.

// Parser      string            `yaml:"parser,omitempty"               json:"parser,omitempty"`       // How to parse the response content from the request.
// Interval   string `yaml:"interval,omitempty"             json:"interval,omitempty"` // For Async Bindings, this specifies the interval on which data sources should be refreshed (if so desired).
// NotIfExpr  string `yaml:"not_if,omitempty"               json:"not_if,omitempty"`   // Do not evaluate the binding if this expression yields a truthy value.
// OnlyIfExpr string `yaml:"only_if,omitempty"              json:"only_if,omitempty"`  // Only evaluate the binding if this expression yields a truthy value.
// Optional   bool   `yaml:"optional,omitempty"             json:"optional,omitempty"` // Whether the request failing will cause a page-wide error or be ignored.
