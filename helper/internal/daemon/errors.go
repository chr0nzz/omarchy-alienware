package daemon

import (
	"errors"

	"github.com/godbus/dbus/v5"

	"github.com/chr0nzz/omarchy-alienware/helper/internal/fan"
	"github.com/chr0nzz/omarchy-alienware/helper/internal/hw"
	"github.com/chr0nzz/omarchy-alienware/helper/internal/kbd"
)

const (
	Interface  = "org.xyzlab.Alienware1"
	BusName    = "org.xyzlab.Alienware1"
	ObjectPath = "/org/xyzlab/Alienware1"

	ErrNameDenied       = Interface + ".Denied"
	ErrNameBadRequest   = Interface + ".BadRequest"
	ErrNameNotSupported = Interface + ".NotSupported"
	ErrNameHwMissing    = Interface + ".HwMissing"
	ErrNameInternal     = Interface + ".Internal"
)

func dbusError(name, msg string) *dbus.Error {
	return dbus.NewError(name, []any{msg})
}

func errDenied(msg string) *dbus.Error {
	return dbusError(ErrNameDenied, msg)
}

func mapError(err error) *dbus.Error {
	if err == nil {
		return nil
	}
	var derr *dbus.Error
	if errors.As(err, &derr) {
		return derr
	}
	var kerr *kbd.Error
	if errors.As(err, &kerr) {
		switch kerr.Code() {
		case kbd.CodeBadRequest:
			return dbusError(ErrNameBadRequest, err.Error())
		case kbd.CodeNoDevice:
			return dbusError(ErrNameHwMissing, err.Error())
		default:
			return dbusError(ErrNameInternal, err.Error())
		}
	}
	switch {
	case errors.Is(err, hw.ErrBadRequest), errors.Is(err, fan.ErrInvalidCurve):
		return dbusError(ErrNameBadRequest, err.Error())
	case errors.Is(err, hw.ErrNotSupported):
		return dbusError(ErrNameNotSupported, err.Error())
	case errors.Is(err, hw.ErrHardwareMissing):
		return dbusError(ErrNameHwMissing, err.Error())
	default:
		return dbusError(ErrNameInternal, err.Error())
	}
}
