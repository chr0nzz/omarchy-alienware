package kbd

import "fmt"

const (
	CodeBadRequest = "bad-request"
	CodeInternal   = "internal"
	CodeNoDevice   = "kbd-not-found"
	CodeBusy       = "kbd-busy"
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
