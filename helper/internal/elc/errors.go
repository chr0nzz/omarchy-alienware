package elc

import "fmt"

const (
	CodeBadRequest   = "bad-request"
	CodeNotSupported = "not-supported"
	CodeInternal     = "internal"
	CodeNoDevice     = "aw-elc-not-found"
	CodeWedged       = "aw-elc-wedged"
	CodeTimeout      = "aw-elc-timeout"
)

type Error struct {
	code string
	msg  string
}

func (e *Error) Error() string { return e.msg }
func (e *Error) Code() string  { return e.code }

func newError(code, format string, args ...any) *Error {
	return &Error{code: code, msg: fmt.Sprintf(format, args...)}
}
