package diecast

type funcExample struct {
	Code   string
	Return interface{}
}

type funcArg struct {
	Name        string
	Type        string
	Description string
	Variadic    bool
	Optional bool
	Default interface{}
	Valid []funcArg
}

type funcDef struct {
	Name      string
	Alias     string
	Summary   string
	Returns   string
	Arguments []funcArg
	Examples  []funcExample
	Function  interface{}
}

type funcGroup struct {
	Name        string
	Description string
	Functions   []funcDef
}

type funcGroups []funcGroup
