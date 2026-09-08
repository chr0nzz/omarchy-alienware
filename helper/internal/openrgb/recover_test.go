package openrgb

import (
	"errors"
	"testing"
)

type fakeResetter struct {
	calls  int
	onCall func()
	err    error
	node   string
}

func (f *fakeResetter) Reset() (string, error) {
	f.calls++
	if f.onCall != nil {
		f.onCall()
	}
	if f.err != nil {
		return "", f.err
	}
	return f.node, nil
}

func TestOpenWithResetRecoversFromZeroZones(t *testing.T) {
	server := newFakeORGBServer(t, zeroZoneController())
	resetter := &fakeResetter{node: "/dev/bus/usb/003/017", onCall: func() {
		server.setCtrl(testController())
	}}
	session, err := OpenWithReset(server.addr(), resetter)
	if err != nil {
		t.Fatalf("OpenWithReset: %v", err)
	}
	t.Cleanup(func() { session.Close() })
	if resetter.calls != 1 {
		t.Fatalf("reset calls: got %d, want exactly 1", resetter.calls)
	}
	if len(session.Ctrl.Zones) == 0 {
		t.Fatal("want zones populated after recovery")
	}
}

func TestOpenWithResetStillZeroAfterReset(t *testing.T) {
	server := newFakeORGBServer(t, zeroZoneController())
	resetter := &fakeResetter{node: "/dev/bus/usb/003/017"}
	_, err := OpenWithReset(server.addr(), resetter)
	if err == nil {
		t.Fatal("want an error when zones stay zero after a reset")
	}
	var oerr *Error
	if !errors.As(err, &oerr) || oerr.Code() != CodeWedged {
		t.Fatalf("got %v, want a %q error", err, CodeWedged)
	}
	if resetter.calls != 1 {
		t.Fatalf("reset calls: got %d, want exactly 1, a wedged device must not loop", resetter.calls)
	}
}

func TestOpenWithResetHealthyDeviceNeverResets(t *testing.T) {
	server := newFakeORGBServer(t, testController())
	resetter := &fakeResetter{node: "/dev/bus/usb/003/017"}
	session, err := OpenWithReset(server.addr(), resetter)
	if err != nil {
		t.Fatalf("OpenWithReset: %v", err)
	}
	t.Cleanup(func() { session.Close() })
	if resetter.calls != 0 {
		t.Fatalf("reset calls: got %d, want 0 for a healthy device", resetter.calls)
	}
}

func TestOpenWithResetDeviceNotFound(t *testing.T) {
	server := newFakeORGBServer(t, zeroZoneController())
	resetter := &fakeResetter{err: errors.New("no AW-ELC controller (187c:0550) found on the USB bus")}
	_, err := OpenWithReset(server.addr(), resetter)
	if err == nil {
		t.Fatal("want an error when the reset cannot find the device")
	}
	var oerr *Error
	if !errors.As(err, &oerr) || oerr.Code() != CodeWedged {
		t.Fatalf("got %v, want a %q error", err, CodeWedged)
	}
	if resetter.calls != 1 {
		t.Fatalf("reset calls: got %d, want exactly 1", resetter.calls)
	}
}
