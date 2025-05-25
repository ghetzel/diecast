package internal

type FuncExample struct {
	Input       any
	Code        string
	Description string
	Return      any
	SkipTest    bool
}

type FuncArg struct {
	Name        string
	Type        string
	Description string
	Variadic    bool
	Optional    bool
	Default     any
	Valid       []FuncArg
}

type FuncDef struct {
	Name      string
	Alias     string
	Aliases   []string
	Summary   string
	Returns   string
	Hidden    bool
	Arguments []FuncArg
	Examples  []FuncExample
	Function  any `json:"-"`
	SkipTest  bool
}

type FuncGroup struct {
	Name        string
	Description string
	Functions   []FuncDef
	Skip        bool
	SkipTest    bool
}

func (self FuncGroup) fn(name string) any {
	for _, fn := range self.Functions {
		if fn.Name == name {
			return fn.Function
		}
	}

	return nil
}

type FuncGroups []FuncGroup

func (self FuncGroups) PopulateFuncMap(funcs FuncMap) {
	for _, group := range self {
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
