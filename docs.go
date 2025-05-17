package diecast

type funcExample struct {
	Code        string
	Description string
	Return      any
}

type funcArg struct {
	Name        string
	Type        string
	Description string
	Variadic    bool
	Optional    bool
	Default     any
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
	Function  any `json:"-"`
}

type funcGroup struct {
	Name        string
	Description string
	Functions   []funcDef
	Skip        bool
}

func (group funcGroup) fn(name string) any {
	for _, fn := range group.Functions {
		if fn.Name == name {
			return fn.Function
		}
	}

	return nil
}

type funcGroups []funcGroup

func (group funcGroups) PopulateFuncMap(funcs FuncMap) {
	for _, group := range group {
		if group.Skip {
			continue
		}

		for _, fn := range group.Functions {
			if fn.Name != `` && fn.Function != nil {
				funcs[fn.Name] = fn.Function
			}
		}
	}
}
