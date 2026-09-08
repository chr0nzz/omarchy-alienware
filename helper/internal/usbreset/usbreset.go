package usbreset

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

const (
	VendorID  = "187c"
	ProductID = "0550"

	CodeNotFound = "aw-elc-not-found"
	CodeFailed   = "aw-elc-reset-failed"

	usbdevfsReset = 0x5514

	defaultSysRoot = "/sys/bus/usb/devices"
	defaultDevRoot = "/dev/bus/usb"
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

type Resetter interface {
	Reset() (string, error)
}

type Locator struct {
	SysRoot string
	DevRoot string
}

func New() *Locator {
	return &Locator{SysRoot: defaultSysRoot, DevRoot: defaultDevRoot}
}

func (l *Locator) sysRoot() string {
	if l.SysRoot == "" {
		return defaultSysRoot
	}
	return l.SysRoot
}

func (l *Locator) devRoot() string {
	if l.DevRoot == "" {
		return defaultDevRoot
	}
	return l.DevRoot
}

func readTrimmed(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func (l *Locator) find() (string, error) {
	entries, err := os.ReadDir(l.sysRoot())
	if err != nil {
		return "", newError(CodeNotFound, "cannot read %s: %s", l.sysRoot(), err.Error())
	}
	for _, entry := range entries {
		dir := filepath.Join(l.sysRoot(), entry.Name())
		if readTrimmed(filepath.Join(dir, "idVendor")) != VendorID {
			continue
		}
		if readTrimmed(filepath.Join(dir, "idProduct")) != ProductID {
			continue
		}
		bus, err := strconv.Atoi(readTrimmed(filepath.Join(dir, "busnum")))
		if err != nil {
			continue
		}
		dev, err := strconv.Atoi(readTrimmed(filepath.Join(dir, "devnum")))
		if err != nil {
			continue
		}
		return filepath.Join(l.devRoot(), fmt.Sprintf("%03d", bus), fmt.Sprintf("%03d", dev)), nil
	}
	return "", newError(CodeNotFound, "no AW-ELC controller (%s:%s) found on the USB bus", VendorID, ProductID)
}

func (l *Locator) Reset() (string, error) {
	node, err := l.find()
	if err != nil {
		return "", err
	}
	fd, err := syscall.Open(node, syscall.O_WRONLY, 0)
	if err != nil {
		return "", newError(CodeFailed, "cannot open %s: %s", node, err.Error())
	}
	defer syscall.Close(fd)
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), uintptr(usbdevfsReset), 0)
	if errno != 0 {
		return "", newError(CodeFailed, "USBDEVFS_RESET on %s failed: %s", node, errno.Error())
	}
	return node, nil
}
