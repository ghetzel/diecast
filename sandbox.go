package diecast

import (
	"fmt"
	"time"

	"github.com/ghetzel/go-stockutil/maputil"
)

type SandboxCommandHandler func(*SandboxMessage) *SandboxMessage

type SandboxMessage struct {
	Command      string        `json:"command"`
	Args         []interface{} `json:"args"`
	Response     interface{}   `json:"response"`
	Missing      bool          `json:"missing"`
	ErrorMessage string        `json:"error"`
	RequestedAt  time.Time     `json:"requested_at"`
	RespondedAt  time.Time     `json:"responded_at"`
}

func (self *SandboxMessage) Error() string {
	return self.ErrorMessage
}

func (self *SandboxMessage) SetError(message error) {
	if message == nil {
		self.ErrorMessage = ``
	} else {
		self.ErrorMessage = message.Error()
	}
}

func (self *SandboxMessage) HasResponded() bool {
	return self.RespondedAt.IsZero()
}

func (self *SandboxMessage) DoneResponding() time.Duration {
	self.RespondedAt = time.Now()
	return self.Duration()
}

func (self *SandboxMessage) Duration() time.Duration {
	return self.RespondedAt.Sub(self.RequestedAt)
}

// This struct is intended to serve as the mainloop in a companion process that is
// used to sandbox and distribute template function evaluation.
type Sandbox struct {
	commandRoutes maputil.Map
}

func (self *Sandbox) Run() error {
	return nil
}

func (self *Sandbox) RegisterHandler(commandName string, handler SandboxCommandHandler) error {
	if handler == nil {
		return fmt.Errorf("cannot use nil command handler")
	}

	self.commandRoutes.Set(commandName, handler)
	return nil
}
