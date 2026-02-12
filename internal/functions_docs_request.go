package internal

import (
	"github.com/ghetzel/go-stockutil/httputil"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/typeutil"
)

func loadRuntimeFunctionsRequest(server ServerProxy, ctx Contextable) FuncGroup {
	var req = ctx.Request()

	if req == nil {
		panic("cannot load request-level functions without an *http.Request object")
	}

	return FuncGroup{
		Name: `HTTP Request Details`,
		Description: `These functions provide access to information contained in the original HTTP client request that ` +
			`led to the current template being processed.  These functions are useful for allowing user-specified data ` +
			`to drive how the output is generated.`,
		Functions: []FuncDef{
			{
				Name:    `payload`,
				Summary: `Return either the request body in its entirety, or (if a key is given), parses the body as a data structure (according to the request Content-Type) and attempts to return the value at that key.`,
				Arguments: []FuncArg{
					{
						Name:        `key`,
						Type:        `string`,
						Optional:    true,
						Description: "The key (may be deeply.nested) to retrieve from the request body after attempting to parse it.",
					},
				},
				Function: func(datakeys ...any) (any, error) {
					if len(datakeys) == 0 {
						return req.Body, nil
					} else {
						var assumeItsAMap map[string]any

						if err := httputil.ParseRequest(req, &assumeItsAMap); err == nil {
							if len(datakeys) == 1 {
								return assumeItsAMap[typeutil.String(datakeys[0])], nil
							} else {
								var values []any

								for _, key := range sliceutil.Stringify(datakeys) {
									values = append(values, assumeItsAMap[key])
								}

								return values, nil
							}
						} else {
							return nil, err
						}
					}
				},
			}, {
				Name:    `cookie`,
				Summary: `Returns the value of a cookie submitted in the request.`,
				Arguments: []FuncArg{
					{
						Name:        `key`,
						Type:        `string`,
						Description: "The name of the cookie value to retrieve.",
					}, {
						Name:        `fallback`,
						Type:        `any`,
						Optional:    true,
						Description: `The value to return of the named cookie is not present or is empty.`,
					},
				},
				Function: func(key any, fallbacks ...any) any {
					if cookie, err := req.Cookie(typeutil.String(key)); err == nil {
						if len(cookie.Value) > 0 {
							return cookie.Value
						}
					}

					if len(fallbacks) > 0 {
						return fallbacks[0]
					} else {
						return nil
					}
				},
			}, {
				Name:    `qs`,
				Summary: `Returns a single querystring value from the request URL.`,
				Arguments: []FuncArg{
					{
						Name:        `key`,
						Type:        `string`,
						Description: "The name of the query string value to retrieve.",
					}, {
						Name:        `fallback`,
						Type:        `any`,
						Optional:    true,
						Description: `The value to return of the named query string is not present or is empty.`,
					},
				},
				Function: func(key any, fallbacks ...any) any {
					return httputil.Q(
						req,
						typeutil.String(key),
						sliceutil.Stringify(fallbacks)...,
					)
				},
			}, {
				Name:    `headers`,
				Summary: `Returns an object containing all HTTP headers in the originating request.`,
				Arguments: []FuncArg{
					{
						Name:        `key`,
						Type:        `string`,
						Description: "The name of the request header value to retrieve.",
					}, {
						Name:        `fallback`,
						Type:        `any`,
						Optional:    true,
						Description: `The value to return of the named request header is not present or is empty.`,
					},
				},
				Function: func(key string, fallbacks ...any) string {
					if v := req.Header.Get(key); v != `` {
						return v
					} else if len(fallbacks) > 0 {
						return typeutil.String(fallbacks[0])
					} else {
						return ``
					}
				},
			}, {
				Name:    `param`,
				Summary: `Returns a positional parameter parsed from the request URL.`,
				Arguments: []FuncArg{
					{
						Name:        `keyOrIndex`,
						Type:        `string`,
						Description: "The name or integral position of the parameter to retrieve",
					}, {
						Name:        `fallback`,
						Type:        `any`,
						Optional:    true,
						Description: `The value to return if the key doesn't exist or is empty.`,
					},
				},
				Function: FunctionNotImplementedError(`param`),
			}, {
				Name:     `read`,
				Function: FunctionNotImplementedError(`read`),
			}, {
				Name:     `i18n`,
				SkipTest: true,
				Summary:  `Return the translation text corresponding to the page's current locale, or from an explicitly-provided locale.`,
				Arguments: []FuncArg{
					{
						Name:        `key`,
						Type:        `string`,
						Description: "The key corresponding to a translated text string in the `translations` section of `diecast.yml` or the page's front matter.",
					}, {
						Name:        `locale`,
						Type:        `string`,
						Optional:    true,
						Description: `Explicitly retrieve a value for the named locale.`,
					},
				},
				Function: FunctionNotImplementedError(`i18n`),
				// Examples: []FuncExample{
				// 	{
				// 		Code:   `i18n "homepage.greeting"`,
				// 		Return: "Hello",
				// 	}, {
				// 		Code:   `i18n "homepage.greeting" "ru"`,
				// 		Return: `Привет`,
				// 	}, {
				// 		Code:   `i18n "homepage.greeting" # browser set to es-EC`,
				// 		Return: `Hola`,
				// 	},
				// },
			},
		},
	}
}
