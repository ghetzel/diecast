package diecast

type funcExample struct {
	Code        string
	Description string
	Return      interface{}
}

type funcArg struct {
	Name        string
	Type        string
	Description string
	Variadic    bool
	Optional    bool
	Default     interface{}
	Valid       []funcArg
}

type funcDef struct {
	Name      string
	Alias     string
	Aliases   []string
	Summary   string
	Returns   string
	Hidden    bool
	Arguments []funcArg
	Examples  []funcExample
	Function  interface{} `json:"-"`
}

type funcGroup struct {
	Name        string
	Description string
	Functions   []funcDef
}

func (self funcGroup) fn(name string) interface{} {
	for _, fn := range self.Functions {
		if fn.Name == name {
			return fn.Function
		}
	}

	return nil
}

type funcGroups []funcGroup

func (self funcGroups) PopulateFuncMap(funcs FuncMap) {
	for _, group := range self {
		for _, fn := range group.Functions {
			if fn.Name != `` && fn.Function != nil {
				funcs[fn.Name] = fn.Function
			}
		}
	}
}
