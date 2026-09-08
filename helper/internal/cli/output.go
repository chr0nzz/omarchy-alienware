package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/chr0nzz/omarchy-alienware/helper/internal/client"
	"github.com/chr0nzz/omarchy-alienware/helper/internal/fan"
	"github.com/chr0nzz/omarchy-alienware/helper/internal/hw"
)

const (
	CodeBadRequest = "bad-request"
	CodeInternal   = "internal"
)

type coded interface {
	Code() string
}

type failure struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
	Code  string `json:"code"`
}

func codeOf(err error) string {
	var c coded
	if errors.As(err, &c) {
		return c.Code()
	}
	switch {
	case errors.Is(err, hw.ErrBadRequest), errors.Is(err, fan.ErrInvalidCurve):
		return client.CodeBadRequest
	case errors.Is(err, hw.ErrNotSupported):
		return client.CodeNotSupported
	case errors.Is(err, hw.ErrHardwareMissing):
		return client.CodeHwMissing
	default:
		return CodeInternal
	}
}

func writeJSON(w io.Writer, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(b))
	return err
}

func emitOK(w io.Writer, v any) int {
	if err := writeJSON(w, v); err != nil {
		return emitError(w, err)
	}
	return 0
}

func emitRaw(w io.Writer, payload string) int {
	fmt.Fprintln(w, payload)
	return 0
}

func emitError(w io.Writer, err error) int {
	_ = writeJSON(w, failure{OK: false, Error: err.Error(), Code: codeOf(err)})
	return 1
}

type usageError struct {
	msg string
}

func (e *usageError) Error() string { return e.msg }
func (e *usageError) Code() string  { return CodeBadRequest }

func badRequest(format string, args ...any) error {
	return &usageError{msg: fmt.Sprintf(format, args...)}
}

type internalError struct {
	msg string
}

func (e *internalError) Error() string { return e.msg }
func (e *internalError) Code() string  { return CodeInternal }

func internalf(format string, args ...any) error {
	return &internalError{msg: fmt.Sprintf(format, args...)}
}
