package hidraw

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

const (
	iocNRBits   = 8
	iocTypeBits = 8
	iocSizeBits = 14

	iocNRShift   = 0
	iocTypeShift = iocNRShift + iocNRBits
	iocSizeShift = iocTypeShift + iocTypeBits
	iocDirShift  = iocSizeShift + iocSizeBits

	iocWrite = 1
	iocRead  = 2

	hidrawType = 'H'
)

const (
	CodeNotFound = "hidraw-not-found"
	CodeIO       = "hidraw-io"
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

func iocNumber(dir, nr, size uintptr) uintptr {
	return dir<<iocDirShift | hidrawType<<iocTypeShift | nr<<iocNRShift | size<<iocSizeShift
}

func hidiocSFeature(size int) uintptr { return iocNumber(iocWrite|iocRead, 0x06, uintptr(size)) }
func hidiocGFeature(size int) uintptr { return iocNumber(iocWrite|iocRead, 0x07, uintptr(size)) }
func hidiocSInput(size int) uintptr   { return iocNumber(iocWrite|iocRead, 0x09, uintptr(size)) }
func hidiocGInput(size int) uintptr   { return iocNumber(iocWrite|iocRead, 0x0A, uintptr(size)) }
func hidiocSOutput(size int) uintptr  { return iocNumber(iocWrite|iocRead, 0x0B, uintptr(size)) }
func hidiocGOutput(size int) uintptr  { return iocNumber(iocWrite|iocRead, 0x0C, uintptr(size)) }
func hidiocGRawInfo() uintptr         { return iocNumber(iocRead, 0x03, 8) }

type DevInfo struct {
	Path    string
	Bus     string
	Vendor  uint16
	Product uint16
	Name    string
}

type RawInfo struct {
	BusType uint32
	Vendor  int16
	Product int16
}

type Device struct {
	file *os.File
	path string
}

func Open(path string) (*Device, error) {
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, newError(CodeNotFound, "hidraw: %s does not exist", path)
		}
		return nil, newError(CodeIO, "hidraw: cannot open %s: %s", path, err.Error())
	}
	return &Device{file: f, path: path}, nil
}

func (d *Device) Close() error { return d.file.Close() }
func (d *Device) Path() string { return d.path }

func doIoctl(fd uintptr, req uintptr, buf []byte) (int, error) {
	if len(buf) == 0 {
		return 0, newError(CodeIO, "hidraw: buffer must not be empty")
	}
	r1, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, req, uintptr(unsafe.Pointer(&buf[0])))
	if errno != 0 {
		return 0, newError(CodeIO, "hidraw: ioctl failed: %s", errno.Error())
	}
	return int(r1), nil
}

func (d *Device) SetFeature(buf []byte) (int, error) {
	return doIoctl(d.file.Fd(), hidiocSFeature(len(buf)), buf)
}

func (d *Device) GetFeature(buf []byte) (int, error) {
	return doIoctl(d.file.Fd(), hidiocGFeature(len(buf)), buf)
}

func (d *Device) SetInput(buf []byte) (int, error) {
	return doIoctl(d.file.Fd(), hidiocSInput(len(buf)), buf)
}

func (d *Device) GetInput(buf []byte) (int, error) {
	return doIoctl(d.file.Fd(), hidiocGInput(len(buf)), buf)
}

func (d *Device) SetOutputReport(buf []byte) (int, error) {
	return doIoctl(d.file.Fd(), hidiocSOutput(len(buf)), buf)
}

func (d *Device) GetOutputReport(buf []byte) (int, error) {
	return doIoctl(d.file.Fd(), hidiocGOutput(len(buf)), buf)
}

func (d *Device) Write(buf []byte) (int, error) {
	n, err := d.file.Write(buf)
	if err != nil {
		return n, newError(CodeIO, "hidraw: write failed: %s", err.Error())
	}
	return n, nil
}

func (d *Device) Read(buf []byte) (int, error) {
	n, err := d.file.Read(buf)
	if err != nil {
		return n, newError(CodeIO, "hidraw: read failed: %s", err.Error())
	}
	return n, nil
}

func (d *Device) RawInfo() (RawInfo, error) {
	buf := make([]byte, 8)
	if _, err := doIoctl(d.file.Fd(), hidiocGRawInfo(), buf); err != nil {
		return RawInfo{}, err
	}
	busType := uint32(buf[0]) | uint32(buf[1])<<8 | uint32(buf[2])<<16 | uint32(buf[3])<<24
	vendor := int16(uint16(buf[4]) | uint16(buf[5])<<8)
	product := int16(uint16(buf[6]) | uint16(buf[7])<<8)
	return RawInfo{BusType: busType, Vendor: vendor, Product: product}, nil
}

func FindByVIDPID(vid, pid uint16) ([]DevInfo, error) {
	const class = "/sys/class/hidraw"
	entries, err := os.ReadDir(class)
	if err != nil {
		return nil, newError(CodeIO, "hidraw: cannot read %s: %s", class, err.Error())
	}
	var matches []DevInfo
	for _, e := range entries {
		ueventPath := filepath.Join(class, e.Name(), "device", "uevent")
		data, err := os.ReadFile(ueventPath)
		if err != nil {
			continue
		}
		info, ok := parseUevent(string(data))
		if !ok {
			continue
		}
		if info.Vendor == vid && info.Product == pid {
			info.Path = filepath.Join("/dev", e.Name())
			matches = append(matches, info)
		}
	}
	sort.Slice(matches, func(i, j int) bool { return matches[i].Path < matches[j].Path })
	return matches, nil
}

func parseUevent(content string) (DevInfo, bool) {
	var info DevInfo
	found := false
	for _, line := range strings.Split(content, "\n") {
		switch {
		case strings.HasPrefix(line, "HID_ID="):
			parts := strings.Split(strings.TrimPrefix(line, "HID_ID="), ":")
			if len(parts) != 3 {
				continue
			}
			vendor, err1 := strconv.ParseUint(parts[1], 16, 32)
			product, err2 := strconv.ParseUint(parts[2], 16, 32)
			if err1 != nil || err2 != nil {
				continue
			}
			info.Bus = parts[0]
			info.Vendor = uint16(vendor)
			info.Product = uint16(product)
			found = true
		case strings.HasPrefix(line, "HID_NAME="):
			info.Name = strings.TrimPrefix(line, "HID_NAME=")
		}
	}
	return info, found
}

func FindOneByVIDPID(vid, pid uint16) (DevInfo, error) {
	matches, err := FindByVIDPID(vid, pid)
	if err != nil {
		return DevInfo{}, err
	}
	if len(matches) == 0 {
		return DevInfo{}, newError(CodeNotFound, "hidraw: no node found for vid=0x%04x pid=0x%04x", vid, pid)
	}
	return matches[0], nil
}
