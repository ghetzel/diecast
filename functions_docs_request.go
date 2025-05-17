package diecast

func loadRuntimeFunctionsRequest(_ *Server) funcGroup {
	return funcGroup{
		Name: `HTTP Request Details`,
		Description: `These functions provide access to information contained in the original HTTP client request that ` +
			`led to the current template being processed.  These functions are useful for allowing user-specified data ` +
			`to drive how the output is generated.`,
		Functions: []funcDef{
			{
				Name:    `payload`,
				Summary: `Return either the request body in its entirety, or (if a key is given), parses the body as a data structure (according to the request Content-Type) and attempts to return the value at that key.`,
				Arguments: []funcArg{
					{
						Name:        `key`,
						Type:        `string`,
						Optional:    true,
						Description: "The key (may be deeply.nested) to retrieve from the request body after attempting to parse it.",
					},
				},
			}, {
				Name:    `querystrings`,
				Summary: `Returns an object containing all querystrings in the request URL.`,
			}, {
				Name:    `cookie`,
				Summary: `Returns the value of a cookie submitted in the request.`,
				Arguments: []funcArg{
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
			}, {
				Name:    `qs`,
				Summary: `Returns a single querystring value from the request URL.`,
				Arguments: []funcArg{
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
			}, {
				Name:    `headers`,
				Summary: `Returns an object containing all HTTP headers in the originating request.`,
				Arguments: []funcArg{
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
			}, {
				Name:    `param`,
				Summary: `Returns a positional parameter parsed from the request URL.`,
				Arguments: []funcArg{
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
			}, {
				Name: `read`,
			}, {
				Name:    `i18n`,
				Summary: `Return the translation text corresponding to the page's current locale, or from an explicitly-provided locale.`,
				Arguments: []funcArg{
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
				Examples: []funcExample{
					{
						Code:   `i18n "homepage.greeting"`,
						Return: "Hello",
					}, {
						Code:   `i18n "homepage.greeting" "ru"`,
						Return: `Привет`,
					}, {
						Code:   `i18n "homepage.greeting" # browser set to es-EC`,
						Return: `Hola`,
					},
				},
			},
		},
	}
}
