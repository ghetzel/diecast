package internal

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
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

// Returns a string representation of the function signature this message represents.
func (self *SandboxMessage) String() string {
	var sig = self.Command + `(`

	if len(self.Args) > 0 {
		var argtypes = make([]string, 0)

		for _, arg := range self.Args {
			argtypes = append(argtypes, reflect.TypeOf(arg).String())
		}

		sig = sig + strings.Join(argtypes, `, `) + `)`
	}

	sig = sig + ` (any, error)`

	return sig
}

func (self *SandboxMessage) Error() string {
	return self.ErrorMessage
}

func (self *SandboxMessage) Err() error {
	if msg := self.Error(); msg != `` {
		return errors.New(msg)
	} else if self.Missing {
		return fmt.Errorf("command %q is not implemented", self.Command)
	} else {
		return nil
	}
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

// Mark the message as having been responded to.
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

// Executes the given command, passing it the supplied arguments in the order they are given.
// Any output from the executed command will be returned, or an error if one occurred.
func (self *Sandbox) Call(commandName string, args ...interface{}) (interface{}, error) {
	if handler, ok := self.commandRoutes.Get(commandName).Value.(SandboxCommandHandler); ok {
		var msg = handler(&SandboxMessage{
			Command:     commandName,
			Args:        args,
			RequestedAt: time.Now(),
		})

		if err := msg.Err(); err == nil {
			return msg.Response, nil
		} else {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("unhandled command %q", commandName)
	}
}
