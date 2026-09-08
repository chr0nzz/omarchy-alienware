package kbd

import (
	"errors"
	"testing"

	"github.com/chr0nzz/omarchy-alienware/helper/internal/hidraw"
)

func withFindDevice(t *testing.T, fn func(vid, pid uint16) (hidraw.DevInfo, error)) {
	t.Helper()
	prev := findDevice
	findDevice = fn
	t.Cleanup(func() { findDevice = prev })
}

func TestPresentFalseWhenNoDeviceFound(t *testing.T) {
	t.Setenv(DeviceEnv, "")
	withFindDevice(t, func(uint16, uint16) (hidraw.DevInfo, error) {
		return hidraw.DevInfo{}, errors.New("not found")
	})
	if Present() {
		t.Fatalf("Present() = true, want false when no device is found")
	}
}

func TestPresentTrueWhenDeviceFound(t *testing.T) {
	t.Setenv(DeviceEnv, "")
	withFindDevice(t, func(vid, pid uint16) (hidraw.DevInfo, error) {
		return hidraw.DevInfo{Path: "/dev/hidraw9", Vendor: vid, Product: pid}, nil
	})
	if !Present() {
		t.Fatalf("Present() = false, want true when a device is found")
	}
}

func TestOpenDefaultUsesEnvOverride(t *testing.T) {
	t.Setenv(DeviceEnv, "/dev/does-not-exist-alienwarectl-test")
	_, err := OpenDefault()
	if err == nil {
		t.Fatalf("expected an error opening a path that does not exist")
	}
	kerr, ok := err.(*Error)
	if !ok {
		t.Fatalf("error %v is not a *kbd.Error", err)
	}
	if kerr.Code() != CodeNoDevice {
		t.Errorf("code = %q, want %q", kerr.Code(), CodeNoDevice)
	}
}

func TestOpenDefaultSurfacesDiscoveryFailure(t *testing.T) {
	t.Setenv(DeviceEnv, "")
	withFindDevice(t, func(uint16, uint16) (hidraw.DevInfo, error) {
		return hidraw.DevInfo{}, errors.New("no bus entries")
	})
	_, err := OpenDefault()
	if err == nil {
		t.Fatalf("expected an error when discovery fails")
	}
	kerr, ok := err.(*Error)
	if !ok {
		t.Fatalf("error %v is not a *kbd.Error", err)
	}
	if kerr.Code() != CodeNoDevice {
		t.Errorf("code = %q, want %q", kerr.Code(), CodeNoDevice)
	}
}

func TestKeyCountMatchesDefaultRange(t *testing.T) {
	if got := KeyCount(); got != 136 {
		t.Fatalf("KeyCount() = %d, want 136", got)
	}
}
