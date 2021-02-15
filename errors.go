package diecast

import (
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

var ErrNotImplemented = ErrorCode(`Not Implemented`, http.StatusNotImplemented)
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
