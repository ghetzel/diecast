package diecast

import (
	"github.com/ghetzel/go-stockutil/maputil"
)

type InProcessRunner struct {
	funcs maputil.Map
}

func (self *InProcessRunner) HandleMessage(req *SandboxMessage) *SandboxMessage {
	defer req.DoneResponding()

	if fn := self.funcs.Get(req.Command); !fn.IsNil() {

	} else {
		req.Missing = true
	}

	return req
}
