package diecast

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/sliceutil"
	"github.com/ghetzel/go-stockutil/stringutil"
	"github.com/ghetzel/go-stockutil/typeutil"
)

// [type=process] Process the output of the previous step by performing a sequence of discrete
//                operations on the data.
// steps:
// - type: process
//   data: sort
//
// - type: process
//   data: rsort
//
// - type: process
//   data:
// 	   do:        'diffuse'
// 	   separator: '.'
// 	   joiner:    '='
//
// -------------------------------------------------------------------------------------------------
type ProcessStep struct{}

func (self *ProcessStep) Perform(config *StepConfig, w http.ResponseWriter, req *http.Request, prev *StepConfig) (interface{}, error) {
	operations := sliceutil.Sliceify(config.Data)
	data := prev.Output

	config.logstep("prev=%v input=%T", prev, data)

	for _, o := range operations {
		operation := maputil.M(nil)
		var otype string

		if typeutil.IsMap(o) {
			operation = maputil.M(o)
			otype = operation.String(`do`)
		} else {
			otype = typeutil.String(o)
		}

		config.logstep("operation=%s", otype)

		switch otype {
		case `sort`, `rsort`:
			if typeutil.IsArray(data) {
				dataS := sliceutil.Sliceify(data)

				sort.Slice(dataS, func(i int, j int) bool {
					if otype == `rsort` {
						return typeutil.String(dataS[i]) > typeutil.String(dataS[j])
					} else {
						return typeutil.String(dataS[i]) < typeutil.String(dataS[j])
					}
				})

				data = dataS
			} else if data == nil {
				return make([]interface{}, 0), nil
			} else {
				return nil, fmt.Errorf("Can only sort arrays, got %T", data)
			}
		case `diffuse`:
			sep := operation.String(`separator`, `.`)
			joiner := operation.String(`joiner`, `=`)
			dataM := make(map[string]interface{})

			if typeutil.IsArray(data) {
				for i, item := range sliceutil.Sliceify(data) {
					if typeutil.IsScalar(item) {
						k, v := stringutil.SplitPair(typeutil.String(item), joiner)
						k = strings.TrimLeft(k, sep)

						if k == `` {
							k = typeutil.String(i)
						}

						dataM[k] = typeutil.Auto(v)
					} else {
						dataM[typeutil.String(i)] = item
					}
				}
			} else if typeutil.IsMap(data) {
				dataM = maputil.M(data).MapNative()
			} else {
				return nil, fmt.Errorf("Can only diffuse arrays or maps, got %T", data)
			}

			if diffused, err := maputil.DiffuseMap(dataM, sep); err == nil {
				data = diffused
			} else {
				return nil, err
			}

		default:
			return nil, fmt.Errorf("Unrecognized process operation %q", otype)
		}
	}

	return data, nil
}
