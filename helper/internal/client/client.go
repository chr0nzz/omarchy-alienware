package client

import (
	"fmt"
	"strings"

	"github.com/godbus/dbus/v5"

	"github.com/chr0nzz/omarchy-alienware/helper/internal/daemon"
)

const (
	CodeNoDaemon     = "no-daemon"
	CodeDenied       = "denied"
	CodeNotSupported = "not-supported"
	CodeBadRequest   = "bad-request"
	CodeHwMissing    = "hw-missing"
	CodeBusy         = "device-busy"
	CodeInternal     = "internal"
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

func MapDBusError(err error) error {
	if err == nil {
		return nil
	}
	derr, ok := err.(dbus.Error)
	if !ok {
		if p, isPtr := err.(*dbus.Error); isPtr && p != nil {
			derr = *p
			ok = true
		}
	}
	if !ok {
		return newError(CodeInternal, "%s", err.Error())
	}
	msg := derr.Error()
	if msg == "" {
		msg = derr.Name
	}
	switch {
	case strings.HasSuffix(derr.Name, ".Denied"):
		return newError(CodeDenied, "%s", msg)
	case strings.HasSuffix(derr.Name, ".BadRequest"):
		return newError(CodeBadRequest, "%s", msg)
	case strings.HasSuffix(derr.Name, ".NotSupported"):
		return newError(CodeNotSupported, "%s", msg)
	case strings.HasSuffix(derr.Name, ".HwMissing"):
		return newError(CodeHwMissing, "%s", msg)
	case strings.HasSuffix(derr.Name, ".Busy"):
		return newError(CodeBusy, "%s", msg)
	case derr.Name == "org.freedesktop.DBus.Error.AccessDenied":
		return newError(CodeDenied, "the bus refused the call, check the D-Bus policy is installed: %s", msg)
	case derr.Name == "org.freedesktop.DBus.Error.ServiceUnknown",
		derr.Name == "org.freedesktop.DBus.Error.NameHasNoOwner",
		derr.Name == "org.freedesktop.DBus.Error.NoReply",
		derr.Name == "org.freedesktop.DBus.Error.NoServer",
		derr.Name == "org.freedesktop.DBus.Error.Disconnected",
		derr.Name == "org.freedesktop.DBus.Error.TimedOut",
		derr.Name == "org.freedesktop.systemd1.NoSuchUnit",
		derr.Name == "org.freedesktop.DBus.Error.Spawn.ChildExited":
		return newError(CodeNoDaemon, "the alienwarectl daemon is not reachable: %s", msg)
	default:
		return newError(CodeInternal, "%s", msg)
	}
}

type Client struct {
	conn *dbus.Conn
	obj  dbus.BusObject
}

func Connect() (*Client, error) {
	conn, err := dbus.SystemBus()
	if err != nil {
		return nil, newError(CodeNoDaemon, "cannot connect to the system bus: %s", err.Error())
	}
	obj := conn.Object(daemon.BusName, dbus.ObjectPath(daemon.ObjectPath))
	return &Client{conn: conn, obj: obj}, nil
}

func (c *Client) call(method string, args ...any) *dbus.Call {
	return c.obj.Call(daemon.Interface+"."+method, 0, args...)
}

func (c *Client) Status() (string, error) {
	var out string
	if err := c.call("Status").Store(&out); err != nil {
		return "", MapDBusError(err)
	}
	return out, nil
}

func (c *Client) SetProfile(name string) error {
	return MapDBusError(c.call("SetProfile", name).Store())
}

func (c *Client) SetBoost(fanID string, value uint32) error {
	return MapDBusError(c.call("SetBoost", fanID, value).Store())
}

func (c *Client) ApplyCurve(payload string) error {
	return MapDBusError(c.call("ApplyCurve", payload).Store())
}

func (c *Client) StopCurve() error {
	return MapDBusError(c.call("StopCurve").Store())
}

func (c *Client) SetTurbo(on bool) error {
	return MapDBusError(c.call("SetTurbo", on).Store())
}

func (c *Client) SetPowerLimit(constraint, watts uint32) error {
	return MapDBusError(c.call("SetPowerLimit", constraint, watts).Store())
}

func (c *Client) KeyboardStatus() (string, error) {
	var out string
	if err := c.call("KeyboardStatus").Store(&out); err != nil {
		return "", MapDBusError(err)
	}
	return out, nil
}

func (c *Client) SetKeyboardKeys(keys string) error {
	return MapDBusError(c.call("SetKeyboardKeys", keys).Store())
}

func (c *Client) SetKeyboardAll(color string) error {
	return MapDBusError(c.call("SetKeyboardAll", color).Store())
}

func (c *Client) KeyboardOff() error {
	return MapDBusError(c.call("KeyboardOff").Store())
}
