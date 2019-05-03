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
				Name: `querystrings`,
			}, {
				Name: `qs`,
			}, {
				Name: `headers`,
			}, {
				Name: `param`,
			}, {
				Name: `read`,
			},
		},
	}
}
