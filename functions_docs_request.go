package diecast

func loadRuntimeFunctionsRequest(server *Server) funcGroup {
	return funcGroup{
		Name: `HTTP Request Details`,
		Description: `These functions provide access to information contained in the original HTTP client request that ` +
			`led to the current template being processed.  These functions are useful for allowing user-specified data ` +
			`to drive how the output is generated.`,
		Functions: []funcDef{
			{
				Name: `payload`,
			}, {
				Name:    `querystrings`,
				Summary: `Returns an object containing all querystrings in the request URL.`,
			}, {
				Name:    `qs`,
				Summary: `Returns a single querystring value from the request URL.`,
			}, {
				Name:    `headers`,
				Summary: `Returns an object containing all HTTP headers in the originating request.`,
			}, {
				Name:    `param`,
				Summary: `Returns a URL parameter from the request URL.`,
			}, {
				Name: `read`,
			}, {
				Name:    `i18n`,
				Summary: `Return the translation text corresponding to the page's current locale, or from an explicitly-provided locale.`,
			},
		},
	}
}
