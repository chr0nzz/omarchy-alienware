package cli

import (
	"strings"
	"testing"

	"github.com/chr0nzz/omarchy-alienware/helper/internal/elc"
	"github.com/chr0nzz/omarchy-alienware/helper/internal/hidraw"
)

type fakeCLIResetter struct {
	calls int
	node  string
	err   error
}

func (f *fakeCLIResetter) Reset() (string, error) {
	f.calls++
	if f.err != nil {
		return "", f.err
	}
	return f.node, nil
}

func withFakeResetter(t *testing.T, fake *fakeCLIResetter) {
	t.Helper()
	prev := rgbResetter
	rgbResetter = fake
	t.Cleanup(func() { rgbResetter = prev })
}

type fakeRGBTransport struct {
	outputs [][]byte
	status  byte
}

func (f *fakeRGBTransport) SetOutputReport(buf []byte) (int, error) {
	f.outputs = append(f.outputs, append([]byte(nil), buf...))
	return len(buf), nil
}

func (f *fakeRGBTransport) Write(buf []byte) (int, error) { return f.SetOutputReport(buf) }

func (f *fakeRGBTransport) GetInput(buf []byte) (int, error) {
	b := make([]byte, elc.ReportLength)
	b[2] = f.status
	return copy(buf, b), nil
}

func (f *fakeRGBTransport) SetFeature(buf []byte) (int, error) { return len(buf), nil }
func (f *fakeRGBTransport) GetFeature(buf []byte) (int, error) { return len(buf), nil }
func (f *fakeRGBTransport) RawInfo() (hidraw.RawInfo, error)   { return hidraw.RawInfo{}, nil }
func (f *fakeRGBTransport) Close() error                       { return nil }
func (f *fakeRGBTransport) Path() string                       { return "/dev/fake-hidraw" }

func withFakeRGBSession(t *testing.T) *fakeRGBTransport {
	t.Helper()
	ft := &fakeRGBTransport{status: elc.StatusV4Ready}
	dev := elc.NewWithTransport(ft, elc.WriteModeOutput)
	session := elc.NewSession(dev, "/dev/fake-hidraw")
	prev := openRGBSessionFunc
	openRGBSessionFunc = func() (*elc.Session, error) { return session, nil }
	t.Cleanup(func() { openRGBSessionFunc = prev })
	return ft
}

func withFailingRGBSession(t *testing.T, err error) {
	t.Helper()
	prev := openRGBSessionFunc
	openRGBSessionFunc = func() (*elc.Session, error) { return nil, err }
	t.Cleanup(func() { openRGBSessionFunc = prev })
}

func TestRGBStatusShape(t *testing.T) {
	withFakeRGBSession(t)
	code, parsed, raw := run(t, "", "rgb", "status")
	if code != 0 {
		t.Fatalf("exit code: got %d, raw=%s", code, raw)
	}
	if parsed["backend"] != "alienfx" {
		t.Fatalf("backend: got %v, raw=%s", parsed["backend"], raw)
	}
	device, ok := parsed["device"].(map[string]any)
	if !ok {
		t.Fatalf("device object missing, raw=%s", raw)
	}
	if device["path"] != "/dev/fake-hidraw" || device["vendorId"] != "0x187c" || device["productId"] != "0x0550" || device["ready"] != true {
		t.Fatalf("device: got %v, raw=%s", device, raw)
	}
	regions, ok := parsed["regions"].([]any)
	if !ok || len(regions) != 4 {
		t.Fatalf("regions: got %v, raw=%s", parsed["regions"], raw)
	}
}

func TestRGBSetReportsRegionAndColor(t *testing.T) {
	ft := withFakeRGBSession(t)
	code, parsed, raw := run(t, "", "rgb", "set", "logo", "ff8800")
	if code != 0 {
		t.Fatalf("exit code: got %d, raw=%s", code, raw)
	}
	if parsed["region"] != "logo" || parsed["color"] != "ff8800" {
		t.Fatalf("got %v, raw=%s", parsed, raw)
	}
	if len(ft.outputs) == 0 {
		t.Fatalf("expected the fake device to receive at least one frame")
	}
}

func TestRGBSetRejectsUnknownRegion(t *testing.T) {
	withFakeRGBSession(t)
	code, parsed, raw := run(t, "", "rgb", "set", "keyboard", "ff8800")
	if code == 0 {
		t.Fatalf("want a non zero exit code, raw=%s", raw)
	}
	if parsed["ok"] != false || parsed["code"] != CodeBadRequest {
		t.Fatalf("got %v, raw=%s", parsed, raw)
	}
	for _, want := range []string{"power", "logo", "ring-top", "ring-bottom"} {
		if !strings.Contains(parsed["error"].(string), want) {
			t.Errorf("error %q does not name valid region %q", parsed["error"], want)
		}
	}
}

func TestRGBSetAllReportsColor(t *testing.T) {
	withFakeRGBSession(t)
	code, parsed, raw := run(t, "", "rgb", "set-all", "00ff00")
	if code != 0 {
		t.Fatalf("exit code: got %d, raw=%s", code, raw)
	}
	if parsed["color"] != "00ff00" {
		t.Fatalf("color: got %v, raw=%s", parsed["color"], raw)
	}
}

func TestRGBSetMapAppliesEveryNamedRegionInOneTransaction(t *testing.T) {
	ft := withFakeRGBSession(t)
	code, parsed, raw := run(t, "", "rgb", "set-map", "logo=ff0000,power=00ff00")
	if code != 0 {
		t.Fatalf("exit code: got %d, raw=%s", code, raw)
	}
	regions, ok := parsed["regions"].(map[string]any)
	if !ok || regions["logo"] != "ff0000" || regions["power"] != "00ff00" {
		t.Fatalf("regions: got %v, raw=%s", parsed["regions"], raw)
	}
	if len(ft.outputs) != 5 {
		t.Fatalf("got %d frames, want 5 for one reset (2), one colour frame per region and one apply, frames: %v", len(ft.outputs), ft.outputs)
	}
}

func TestRGBSetMapRejectsUnknownRegion(t *testing.T) {
	withFakeRGBSession(t)
	code, parsed, raw := run(t, "", "rgb", "set-map", "keyboard=ff0000")
	if code == 0 {
		t.Fatalf("want a non zero exit code, raw=%s", raw)
	}
	if parsed["ok"] != false || parsed["code"] != CodeBadRequest {
		t.Fatalf("got %v, raw=%s", parsed, raw)
	}
}

func TestRGBSetMapRejectsMalformedEntry(t *testing.T) {
	withFakeRGBSession(t)
	code, parsed, raw := run(t, "", "rgb", "set-map", "logo-ff0000")
	if code == 0 {
		t.Fatalf("want a non zero exit code, raw=%s", raw)
	}
	if parsed["ok"] != false || parsed["code"] != CodeBadRequest {
		t.Fatalf("got %v, raw=%s", parsed, raw)
	}
}

func TestRGBModeReturnsNotSupported(t *testing.T) {
	withFakeRGBSession(t)
	code, parsed, raw := run(t, "", "rgb", "mode", "spectrum")
	if code == 0 {
		t.Fatalf("want a non zero exit code, raw=%s", raw)
	}
	if parsed["ok"] != false || parsed["code"] != "not-supported" {
		t.Fatalf("got %v, raw=%s", parsed, raw)
	}
}

func TestRGBBrightnessReportsValue(t *testing.T) {
	withFakeRGBSession(t)
	code, parsed, raw := run(t, "", "rgb", "brightness", "75")
	if code != 0 {
		t.Fatalf("exit code: got %d, raw=%s", code, raw)
	}
	if parsed["brightness"] != float64(75) {
		t.Fatalf("brightness: got %v, raw=%s", parsed["brightness"], raw)
	}
}

func TestRGBBrightnessRejectsOutOfRange(t *testing.T) {
	withFakeRGBSession(t)
	code, parsed, raw := run(t, "", "rgb", "brightness", "150")
	if code == 0 {
		t.Fatalf("want a non zero exit code, raw=%s", raw)
	}
	if parsed["code"] != CodeBadRequest {
		t.Fatalf("code: got %v, raw=%s", parsed["code"], raw)
	}
}

func TestRGBIdentifyReportsRegion(t *testing.T) {
	withFakeRGBSession(t)
	code, parsed, raw := run(t, "", "rgb", "identify", "ring-top")
	if code != 0 {
		t.Fatalf("exit code: got %d, raw=%s", code, raw)
	}
	if parsed["region"] != "ring-top" {
		t.Fatalf("region: got %v, raw=%s", parsed["region"], raw)
	}
}

func TestRGBOffSucceeds(t *testing.T) {
	withFakeRGBSession(t)
	code, parsed, raw := run(t, "", "rgb", "off")
	if code != 0 {
		t.Fatalf("exit code: got %d, raw=%s", code, raw)
	}
	if parsed["ok"] != true {
		t.Fatalf("got %v, raw=%s", parsed, raw)
	}
}

func TestRGBResetReportsNode(t *testing.T) {
	fake := &fakeCLIResetter{node: "/dev/bus/usb/003/017"}
	withFakeResetter(t, fake)
	code, parsed, raw := run(t, "", "rgb", "reset")
	if code != 0 {
		t.Fatalf("exit code: got %d, raw=%s", code, raw)
	}
	if parsed["node"] != "/dev/bus/usb/003/017" {
		t.Fatalf("node: got %v, raw=%s", parsed["node"], raw)
	}
	if fake.calls != 1 {
		t.Fatalf("reset calls: got %d, want 1", fake.calls)
	}
}

func TestRGBResetReportsDeviceNotFound(t *testing.T) {
	fake := &fakeCLIResetter{err: errNoAWELC}
	withFakeResetter(t, fake)
	code, parsed, raw := run(t, "", "rgb", "reset")
	if code == 0 {
		t.Fatalf("want a non zero exit code, raw=%s", raw)
	}
	if parsed["ok"] != false {
		t.Fatalf("got %v, raw=%s", parsed, raw)
	}
}

func TestRGBResetSkipsOpeningTheDevice(t *testing.T) {
	fake := &fakeCLIResetter{node: "/dev/bus/usb/003/017"}
	withFakeResetter(t, fake)
	withFailingRGBSession(t, errNoAWELC)
	code, _, raw := run(t, "", "rgb", "reset")
	if code != 0 {
		t.Fatalf("rgb reset must not require opening the AW-ELC device, exit code %d, raw=%s", code, raw)
	}
}

func TestRGBCommandsFailWhenTheDeviceCannotBeOpened(t *testing.T) {
	withFailingRGBSession(t, errNoAWELC)
	code, parsed, raw := run(t, "", "rgb", "status")
	if code == 0 {
		t.Fatalf("want a non zero exit code, raw=%s", raw)
	}
	if parsed["ok"] != false {
		t.Fatalf("got %v, raw=%s", parsed, raw)
	}
}

var errNoAWELC = &fakeCoded{msg: "no AW-ELC controller (187c:0550) found on the USB bus", code: "aw-elc-not-found"}
