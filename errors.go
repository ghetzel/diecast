package diecast

import (
	"fmt"
	"net/http"
)

type CodeableError struct {
	msg  string
	code int
}

func (self *CodeableError) Code() int {
	if self.code == 0 {
		return http.StatusInternalServerError
	} else {
		return self.code
	}
}

func (self *CodeableError) Error() string {
	return self.msg
}

func ErrorCode(msg string, code int) error {
	return &CodeableError{
		msg:  msg,
		code: code,
	}
}

var NotImplemented = func(msg interface{}) error {
	if s, ok := msg.(string); ok && s != `` {
		return ErrorCode(`Not Implemented: `+s, http.StatusNotImplemented)
	} else if err, ok := msg.(error); ok && err != nil {
		return ErrorCode(`Not Implemented: `+err.Error(), http.StatusNotImplemented)
	} else {
		return ErrorCode(`Not Implemented: `+fmt.Sprintf("%T", msg), http.StatusNotImplemented)
	}
}

var ErrNotFound = ErrorCode(`no such file or directory`, http.StatusNotFound)

type ControlError struct {
	message string
}

func (self *ControlError) Error() string {
	if self.message == `` {
		return `stop`
	} else {
		return self.message
	}
}
