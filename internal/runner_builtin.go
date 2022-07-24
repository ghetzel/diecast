package internal

import (
	"fmt"
	"reflect"

	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/typeutil"
)

type InProcessRunner struct {
	funcs maputil.Map
}

func (self *InProcessRunner) Init() error {
	for name, fn := range BuiltinFunctions {
		self.SetFunction(name, fn)
	}

	return nil
}

func (self *InProcessRunner) SetFunction(name string, fn interface{}) {
	self.funcs.Set(name, fn)
}

func (self *InProcessRunner) HandleMessage(req *SandboxMessage) *SandboxMessage {
	defer req.DoneResponding()

	if fn := self.funcs.Get(req.Command); fn.IsFunction(req.String()) {
		var fnV = reflect.ValueOf(fn.Value)

		if output, err := callGoFunction(fnV, req.Args...); err == nil {
			req.Response = output
		} else {
			req.SetError(err)
		}
	} else {
		req.Missing = true
	}

	return req
}

func callGoFunction(fn reflect.Value, inputs ...interface{}) (interface{}, error) {
	var arguments = make([]reflect.Value, fn.Type().NumIn())

	// loop through the arguments the target function takes, building an equally-sized list
	// of reflect.Value instances containing the Golang value we work out using various magicks.
	for i := 0; i < len(arguments); i++ {
		var argT = fn.Type().In(i)

		// first and foremost, initialize the argument to its zero value
		arguments[i] = reflect.Zero(argT)

		// if we received a valid input for this argument, populate it
		if i < len(inputs) {
			if inV := reflect.ValueOf(inputs[i]); inV.IsValid() {
				if inV.Type().AssignableTo(argT) {
					// attempt direct assignment
					arguments[i] = inV
					continue
				} else if inV.Type().ConvertibleTo(argT) {
					// attempt type conversion
					arguments[i] = inV.Convert(argT)
					continue
				}

				// dereference pointers
				if argT.Kind() == reflect.Ptr {
					argT = argT.Elem()
				}

				// instantiate new arg type
				if typeutil.IsScalar(argT) {
					arguments[i] = reflect.Zero(argT)
				} else {
					arguments[i] = reflect.New(argT)
				}

				// map arguments are used to populate newly instantiated structs
				if typeutil.IsMap(inputs[i]) {
					if argT.Kind() == reflect.Struct {
						var inputM = maputil.DeepCopy(inputs[i])

						if len(inputM) > 0 && arguments[i].IsValid() {
							if err := maputil.TaggedStructFromMap(inputM, arguments[i], `json`); err != nil {
								return nil, fmt.Errorf("Cannot populate %v: %v", arguments[i].Type(), err)
							}
						}
					} else {
						return nil, fmt.Errorf("Map arguments can only be used to populate structs")
					}
				}
			}
		}
	}

	// NOTE: it happens here.
	var returns = fn.Call(arguments)

	switch len(returns) {
	case 2:
		if lastT := returns[1].Type(); lastT.Implements(errorInterface) {
			var value = returns[0].Interface()
			var err error

			if v2 := returns[1].Interface(); v2 == nil {
				err = nil
			} else {
				err = v2.(error)
			}

			return value, err
		} else {
			return nil, fmt.Errorf("last return value must be an error, got %v", lastT)
		}

	case 1:
		if lastT := returns[0].Type(); lastT.Implements(errorInterface) {
			if v1 := returns[0].Interface(); v1 == nil {
				return nil, nil
			} else {
				return nil, v1.(error)
			}
		} else {
			return nil, fmt.Errorf("functions returning a single value must return an error, got %v", lastT)
		}
	}

	return nil, nil
}
