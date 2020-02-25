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
// -------------------------------------------------------------------------------------------------
type ProcessStep struct{}

func (self *ProcessStep) Perform(config *StepConfig, w http.ResponseWriter, req *http.Request, prev *StepConfig) (interface{}, error) {
	var operations = sliceutil.Sliceify(config.Data)
	var data = prev.Output

	config.logstep("prev=%v input=%T", prev, data)

	for _, o := range operations {
		var operation = maputil.M(nil)
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
				var dataS = sliceutil.Sliceify(data)

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
			var sep = operation.String(`separator`, `.`)
			var joiner = operation.String(`joiner`, `=`)
			var dataM = make(map[string]interface{})

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
		case `join`:
			var sep = operation.String(`separator`, "\n")
			var kvjoin = operation.String(`joiner`, "=")
			var lines []string

			if typeutil.IsArray(data) {
				for _, item := range sliceutil.Sliceify(data) {
					if typeutil.IsMap(item) {
						var l = maputil.Join(item, kvjoin, sep)
						lines = append(lines, strings.Split(l, sep)...)
					} else if typeutil.IsScalar(item) {
						lines = append(lines, typeutil.String(item))
					}
				}
			} else if typeutil.IsMap(data) {
				return maputil.Join(maputil.M(data).MapNative(), kvjoin, sep), nil
			}

			return strings.Join(lines, sep), nil
		default:
			return nil, fmt.Errorf("Unrecognized process operation %q", otype)
		}
	}

	return data, nil
}
