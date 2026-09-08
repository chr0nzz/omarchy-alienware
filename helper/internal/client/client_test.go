package client

import (
	"errors"
	"testing"

	"github.com/godbus/dbus/v5"
)

func codeFor(t *testing.T, err error) string {
	t.Helper()
	var e *Error
	if !errors.As(err, &e) {
		t.Fatalf("got %T %v, want *client.Error", err, err)
	}
	return e.Code()
}

func TestMapDBusErrorNames(t *testing.T) {
	cases := map[string]string{
		"org.xyzlab.Alienware1.Denied":                 CodeDenied,
		"org.xyzlab.Alienware1.BadRequest":             CodeBadRequest,
		"org.xyzlab.Alienware1.NotSupported":           CodeNotSupported,
		"org.xyzlab.Alienware1.HwMissing":              CodeHwMissing,
		"org.freedesktop.PolicyKit1.Error.Denied":      CodeDenied,
		"org.freedesktop.DBus.Error.ServiceUnknown":    CodeNoDaemon,
		"org.freedesktop.DBus.Error.NameHasNoOwner":    CodeNoDaemon,
		"org.freedesktop.DBus.Error.NoReply":           CodeNoDaemon,
		"org.freedesktop.DBus.Error.Disconnected":      CodeNoDaemon,
		"org.freedesktop.DBus.Error.Spawn.ChildExited": CodeNoDaemon,
		"org.freedesktop.DBus.Error.Failed":            CodeInternal,
	}
	for name, want := range cases {
		err := MapDBusError(dbus.Error{Name: name, Body: []any{"boom"}})
		if got := codeFor(t, err); got != want {
			t.Errorf("%s: got %q, want %q", name, got, want)
		}
	}
}

func TestMapDBusErrorPointer(t *testing.T) {
	err := MapDBusError(dbus.NewError("org.xyzlab.Alienware1.Denied", []any{"nope"}))
	if got := codeFor(t, err); got != CodeDenied {
		t.Fatalf("got %q, want %q", got, CodeDenied)
	}
}

func TestMapDBusErrorPlainError(t *testing.T) {
	err := MapDBusError(errors.New("something else"))
	if got := codeFor(t, err); got != CodeInternal {
		t.Fatalf("got %q, want %q", got, CodeInternal)
	}
}

func TestMapDBusErrorNil(t *testing.T) {
	if err := MapDBusError(nil); err != nil {
		t.Fatalf("got %v, want nil", err)
	}
}

func TestErrorCarriesMessage(t *testing.T) {
	err := MapDBusError(dbus.Error{Name: "org.xyzlab.Alienware1.BadRequest", Body: []any{"boost 900 out of range"}})
	if err.Error() != "boost 900 out of range" {
		t.Fatalf("message: got %q", err.Error())
	}
}

func TestBusPolicyRefusalIsADenial(t *testing.T) {
	err := MapDBusError(dbus.NewError("org.freedesktop.DBus.Error.AccessDenied", []interface{}{"rejected"}))
	if code := codeFor(t, err); code != CodeDenied {
		t.Fatalf("a bus policy refusal should read as denied, got %q", code)
	}
}

func TestActivationFailuresReadAsNoDaemon(t *testing.T) {
	for _, name := range []string{
		"org.freedesktop.DBus.Error.TimedOut",
		"org.freedesktop.systemd1.NoSuchUnit",
	} {
		err := MapDBusError(dbus.NewError(name, []interface{}{"nope"}))
		if code := codeFor(t, err); code != CodeNoDaemon {
			t.Fatalf("%s should read as no-daemon, got %q", name, code)
		}
	}
}
