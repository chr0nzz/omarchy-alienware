package elc

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/chr0nzz/omarchy-alienware/helper/internal/hidraw"
)

type fakeResetter struct {
	calls int
	node  string
	err   error
}

func (f *fakeResetter) Reset() (string, error) {
	f.calls++
	if f.err != nil {
		return "", f.err
	}
	return f.node, nil
}

func fakeSessionOpener(statuses ...uint8) func() (*Session, error) {
	call := 0
	return func() (*Session, error) {
		status := statuses[call]
		if call < len(statuses)-1 {
			call++
		}
		ft := &fakeTransport{inputResponses: [][]byte{statusFrame(status)}, inputRepeatLast: true}
		dev := NewWithTransport(ft, WriteModeOutput)
		return NewSession(dev, "/dev/fake-hidraw"), nil
	}
}

func TestOpenWithResetSkipsResetWhenNotWedged(t *testing.T) {
	opener := fakeSessionOpener(StatusV4Ready)
	resetter := &fakeResetter{node: "/dev/bus/usb/003/017"}
	session, err := openWithReset(opener, resetter)
	if err != nil {
		t.Fatalf("openWithReset() error = %v", err)
	}
	defer session.Close()
	if resetter.calls != 0 {
		t.Fatalf("reset calls = %d, want 0 when the device is not wedged", resetter.calls)
	}
}

func TestOpenWithResetRecoversFromWedge(t *testing.T) {
	opener := fakeSessionOpener(0, StatusV4Ready)
	resetter := &fakeResetter{node: "/dev/bus/usb/003/017"}
	session, err := openWithReset(opener, resetter)
	if err != nil {
		t.Fatalf("openWithReset() error = %v", err)
	}
	defer session.Close()
	if resetter.calls != 1 {
		t.Fatalf("reset calls = %d, want 1", resetter.calls)
	}
}

func TestOpenWithResetFailsWhenStillWedgedAfterReset(t *testing.T) {
	opener := fakeSessionOpener(0, 0)
	resetter := &fakeResetter{node: "/dev/bus/usb/003/017"}
	_, err := openWithReset(opener, resetter)
	if err == nil {
		t.Fatalf("expected an error when the controller is still wedged after a reset")
	}
	eerr, ok := err.(*Error)
	if !ok {
		t.Fatalf("error %v is not an *elc.Error", err)
	}
	if eerr.Code() != CodeWedged {
		t.Errorf("code = %q, want %q", eerr.Code(), CodeWedged)
	}
}

func TestOpenWithResetFailsWhenNoResetterConfigured(t *testing.T) {
	opener := fakeSessionOpener(0)
	_, err := openWithReset(opener, nil)
	if err == nil {
		t.Fatalf("expected an error when wedged with no resetter")
	}
	eerr, ok := err.(*Error)
	if !ok {
		t.Fatalf("error %v is not an *elc.Error", err)
	}
	if eerr.Code() != CodeWedged {
		t.Errorf("code = %q, want %q", eerr.Code(), CodeWedged)
	}
}

func TestSessionSetMapAppliesEveryNamedRegionInOneTransaction(t *testing.T) {
	ft := &fakeTransport{inputResponses: [][]byte{statusFrame(StatusV4Ready)}}
	dev := NewWithTransport(ft, WriteModeOutput)
	session := NewSession(dev, "/dev/fake-hidraw")

	power, _ := ResolveRegion("power")
	logo, _ := ResolveRegion("logo")
	entries := []RegionColor{
		{Region: logo, R: 0xff, G: 0x88, B: 0x00},
		{Region: power, R: 0x00, G: 0xff, B: 0x00},
	}
	if err := session.SetMap(entries); err != nil {
		t.Fatalf("SetMap() error = %v", err)
	}
	if len(ft.outputs) != 5 {
		t.Fatalf("got %d frames, want 5 (one reset pair, one colour frame per region, one apply), frames: %v", len(ft.outputs), ft.outputs)
	}
	if !bytes.Equal(ft.outputs[0], ControlFrame(ControlRemove, ControlIDCommon)) {
		t.Errorf("expected exactly one remove control frame at position 0")
	}
	if !bytes.Equal(ft.outputs[1], ControlFrame(ControlStartNew, ControlIDCommon)) {
		t.Errorf("expected exactly one start-new control frame at position 1")
	}
	if !bytes.Equal(ft.outputs[len(ft.outputs)-1], ControlFrame(ControlFinishPlay, ControlIDCommon)) {
		t.Errorf("expected exactly one finish-play control frame last")
	}
	if !bytes.Equal(ft.outputs[2], SetOneColorFrame(0xff, 0x88, 0x00, logo.ZoneIDs)) {
		t.Errorf("logo colour frame missing or wrong")
	}
	if !bytes.Equal(ft.outputs[3], SetOneColorFrame(0x00, 0xff, 0x00, power.ZoneIDs)) {
		t.Errorf("power colour frame missing or wrong")
	}
}

func TestSessionSetMapRejectsEmptyList(t *testing.T) {
	ft := &fakeTransport{}
	dev := NewWithTransport(ft, WriteModeOutput)
	session := NewSession(dev, "/dev/fake-hidraw")
	if err := session.SetMap(nil); err == nil {
		t.Fatalf("expected an error for an empty set-map list")
	}
	if len(ft.outputs) != 0 {
		t.Fatalf("no frames should be sent for a rejected set-map call")
	}
}

func TestSessionSetBrightnessRejectsOutOfRange(t *testing.T) {
	ft := &fakeTransport{}
	dev := NewWithTransport(ft, WriteModeOutput)
	session := NewSession(dev, "/dev/fake-hidraw")
	if err := session.SetBrightness(101); err == nil {
		t.Fatalf("expected an error for a brightness above 100")
	}
	if err := session.SetBrightness(-1); err == nil {
		t.Fatalf("expected an error for a negative brightness")
	}
	if len(ft.outputs) != 0 {
		t.Fatalf("no frames should be sent for a rejected brightness call")
	}
}

func TestSessionIdentifyBlinksThenLeavesRegionOff(t *testing.T) {
	ft := &fakeTransport{inputResponses: [][]byte{statusFrame(StatusV4Ready)}, inputRepeatLast: true}
	dev := NewWithTransport(ft, WriteModeOutput)
	session := NewSession(dev, "/dev/fake-hidraw")
	logo, _ := ResolveRegion("logo")

	if err := session.Identify(logo, 1, time.Millisecond); err != nil {
		t.Fatalf("Identify() error = %v", err)
	}
	if len(ft.outputs) != 8 {
		t.Fatalf("got %d frames, want 8: one blink on and one blink off, each a reset (2), a colour frame and an apply, frames: %v", len(ft.outputs), ft.outputs)
	}
	last := ft.outputs[len(ft.outputs)-2]
	if !bytes.Equal(last, SetOneColorFrame(0, 0, 0, logo.ZoneIDs)) {
		t.Errorf("last colour frame = % x, want the region blanked back to black", last)
	}
}

func TestSessionStatusListsEveryRegion(t *testing.T) {
	ft := &fakeTransport{}
	dev := NewWithTransport(ft, WriteModeOutput)
	session := NewSession(dev, "/dev/fake-hidraw")
	device, regions := session.Status()
	if device.Path != "/dev/fake-hidraw" || !device.Ready {
		t.Fatalf("device status = %+v", device)
	}
	if len(regions) != len(Regions) {
		t.Fatalf("got %d regions, want %d", len(regions), len(Regions))
	}
}

func unprobedSession(inputs ...[]byte) (*Session, *fakeTransport) {
	ft := &fakeTransport{inputResponses: inputs}
	dev := NewWithTransport(ft, WriteModeOutput)
	return &Session{dev: dev, path: "/dev/fake-hidraw", productID: DefaultProductID}, ft
}

func TestSessionStatusReadinessFollowsTheWedgeProbe(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  bool
	}{
		{name: "ready when the controller answers", input: statusFrame(StatusV4Ready), want: true},
		{name: "not ready when the controller reports all zero", input: statusFrame(0), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session, _ := unprobedSession(tt.input)
			device, _ := session.Status()
			if device.Ready != tt.want {
				t.Fatalf("Ready = %v, want %v", device.Ready, tt.want)
			}
		})
	}
}

func TestSessionStatusIsNotReadyWhenTheProbeFails(t *testing.T) {
	session, _ := unprobedSession()
	device, _ := session.Status()
	if device.Ready {
		t.Fatalf("Ready = true, want false when the probe read fails")
	}
}

func TestSessionStatusProbesAtMostOnce(t *testing.T) {
	session, ft := unprobedSession(statusFrame(StatusV4Ready))
	for i := 0; i < 3; i++ {
		device, _ := session.Status()
		if !device.Ready {
			t.Fatalf("Ready = false on call %d, want the memoised reading", i)
		}
	}
	if ft.inputIdx != 1 {
		t.Fatalf("probe reads = %d, want 1", ft.inputIdx)
	}
}

func TestSessionStatusReusesTheProbeFromOpen(t *testing.T) {
	opener := func() (*Session, error) {
		ft := &fakeTransport{inputResponses: [][]byte{statusFrame(StatusV4Ready)}}
		dev := NewWithTransport(ft, WriteModeOutput)
		return &Session{dev: dev, path: "/dev/fake-hidraw", productID: DefaultProductID}, nil
	}
	session, err := openWithReset(opener, &fakeResetter{})
	if err != nil {
		t.Fatalf("openWithReset() error = %v", err)
	}
	defer session.Close()
	device, _ := session.Status()
	if !device.Ready {
		t.Fatalf("Ready = false, want the reading openWithReset already took with no extra read")
	}
}

func TestDevicePathFallsBackToTheSecondProductID(t *testing.T) {
	tests := []struct {
		name    string
		found   uint16
		wantPID uint16
		wantErr bool
	}{
		{name: "primary id enumerates", found: DefaultProductID, wantPID: DefaultProductID},
		{name: "alternate id enumerates", found: AltProductID, wantPID: AltProductID},
		{name: "neither id enumerates", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(DeviceEnv, "")
			var tried []uint16
			restore := findDevice
			findDevice = func(vid, pid uint16) (hidraw.DevInfo, error) {
				tried = append(tried, pid)
				if pid != tt.found {
					return hidraw.DevInfo{}, errors.New("hidraw: no node found")
				}
				return hidraw.DevInfo{Path: "/dev/fake-hidraw", Vendor: vid, Product: pid}, nil
			}
			defer func() { findDevice = restore }()

			path, pid, err := devicePath()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("devicePath() error = nil, want a no-device error")
				}
				eerr, ok := err.(*Error)
				if !ok {
					t.Fatalf("error %v is not an *elc.Error", err)
				}
				if eerr.Code() != CodeNoDevice {
					t.Errorf("code = %q, want %q", eerr.Code(), CodeNoDevice)
				}
				for _, want := range []string{"0x0550", "0x0551"} {
					if !strings.Contains(eerr.Error(), want) {
						t.Errorf("error %q does not mention %s", eerr.Error(), want)
					}
				}
			} else if err != nil {
				t.Fatalf("devicePath() error = %v", err)
			}
			if !tt.wantErr {
				if path != "/dev/fake-hidraw" {
					t.Errorf("path = %q, want /dev/fake-hidraw", path)
				}
				if pid != tt.wantPID {
					t.Errorf("product id = 0x%04x, want 0x%04x", pid, tt.wantPID)
				}
			}
			if tt.found == AltProductID || tt.wantErr {
				if len(tried) != 2 || tried[0] != DefaultProductID || tried[1] != AltProductID {
					t.Errorf("tried = %v, want 0x0550 then 0x0551", tried)
				}
			}
			if tt.found == DefaultProductID && len(tried) != 1 {
				t.Errorf("tried = %v, want only the primary id", tried)
			}
		})
	}
}

func TestSessionStatusReportsTheProductIDItFound(t *testing.T) {
	ft := &fakeTransport{}
	dev := NewWithTransport(ft, WriteModeOutput)
	session := &Session{dev: dev, path: "/dev/fake-hidraw", productID: AltProductID, probed: true, ready: true}
	device, _ := session.Status()
	if device.ProductID != "0x0551" {
		t.Fatalf("ProductID = %q, want 0x0551", device.ProductID)
	}
}

func TestOpenWithResetSurfacesOpenerError(t *testing.T) {
	sentinel := newError(CodeNoDevice, "no device")
	opener := func() (*Session, error) { return nil, sentinel }
	_, err := openWithReset(opener, &fakeResetter{})
	if !errors.Is(err, sentinel) {
		t.Fatalf("openWithReset() error = %v, want %v", err, sentinel)
	}
}
