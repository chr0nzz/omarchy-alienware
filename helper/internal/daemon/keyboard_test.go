package daemon

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/godbus/dbus/v5"

	"github.com/chr0nzz/omarchy-alienware/helper/internal/hidraw"
	"github.com/chr0nzz/omarchy-alienware/helper/internal/kbd"
)

type fakeKbdTransport struct {
	mu       sync.Mutex
	features [][]byte
	closed   int
}

func (f *fakeKbdTransport) SetFeature(buf []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.features = append(f.features, append([]byte(nil), buf...))
	return len(buf), nil
}

func (f *fakeKbdTransport) GetFeature(buf []byte) (int, error) { return len(buf), nil }
func (f *fakeKbdTransport) RawInfo() (hidraw.RawInfo, error)   { return hidraw.RawInfo{}, nil }
func (f *fakeKbdTransport) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed++
	return nil
}
func (f *fakeKbdTransport) Path() string { return "/dev/fake-hidraw" }

func (f *fakeKbdTransport) frames() [][]byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([][]byte(nil), f.features...)
}

func withFakeKeyboard(t *testing.T, svc *Service) *fakeKbdTransport {
	t.Helper()
	ft := &fakeKbdTransport{}
	svc.openKeyboard = func() (*kbd.Device, error) {
		return kbd.NewWithTransport(ft), nil
	}
	svc.keyboardPresent = func() bool { return true }
	return ft
}

func withAbsentKeyboard(svc *Service) {
	svc.openKeyboard = func() (*kbd.Device, error) {
		return nil, errors.New("no keyboard controller found")
	}
	svc.keyboardPresent = func() bool { return false }
}

func TestKeyboardStatusPresent(t *testing.T) {
	svc := NewService(fakeReader(t), nil, quietLogger())
	withFakeKeyboard(t, svc)

	payload, derr := svc.KeyboardStatus()
	if derr != nil {
		t.Fatalf("unexpected error: %v", derr)
	}
	var out KeyboardStatus
	if err := json.Unmarshal([]byte(payload), &out); err != nil {
		t.Fatalf("status is not valid JSON: %v", err)
	}
	if !out.OK || !out.Present || out.KeyCount != 136 {
		t.Fatalf("got %+v", out)
	}
}

func TestKeyboardStatusAbsentReportsCleanly(t *testing.T) {
	svc := NewService(fakeReader(t), nil, quietLogger())
	withAbsentKeyboard(svc)

	payload, derr := svc.KeyboardStatus()
	if derr != nil {
		t.Fatalf("absence must not be an error, got %v", derr)
	}
	var out KeyboardStatus
	if err := json.Unmarshal([]byte(payload), &out); err != nil {
		t.Fatal(err)
	}
	if !out.OK || out.Present {
		t.Fatalf("got %+v, want present false", out)
	}
	if out.KeyCount != 0 {
		t.Fatalf("keyCount = %d, want omitted/zero when absent", out.KeyCount)
	}
}

func TestSetKeyboardKeysAppliesInOneTransaction(t *testing.T) {
	svc := NewService(fakeReader(t), nil, quietLogger())
	ft := withFakeKeyboard(t, svc)

	if derr := svc.SetKeyboardKeys("0=ff0000,4=00ff00", ":1.1"); derr != nil {
		t.Fatalf("unexpected error: %v", derr)
	}
	frames := ft.frames()
	if len(frames) != 4 {
		t.Fatalf("got %d feature writes, want 4 (reset, color set, loop, update), frames=%v", len(frames), frames)
	}
	want := kbd.ColorSetFrame([]kbd.KeyColor{
		{Index: 0, R: 0xff, G: 0x00, B: 0x00},
		{Index: 4, R: 0x00, G: 0xff, B: 0x00},
	})
	if !bytes.Equal(frames[1], want) {
		t.Errorf("color set frame did not match the requested keys, got % x, want % x", frames[1], want)
	}
	if ft.closed != 1 {
		t.Fatalf("device closed %d times, want exactly 1", ft.closed)
	}
}

func TestSetKeyboardKeysRejectsMalformedPayloadWithoutOpeningTheDevice(t *testing.T) {
	svc := NewService(fakeReader(t), nil, quietLogger())
	opened := false
	svc.openKeyboard = func() (*kbd.Device, error) {
		opened = true
		return nil, errors.New("should not be called")
	}

	derr := svc.SetKeyboardKeys("not-valid", ":1.1")
	if derr == nil {
		t.Fatal("want a rejection")
	}
	if derr.Name != ErrNameBadRequest {
		t.Fatalf("error name: got %q, want %q", derr.Name, ErrNameBadRequest)
	}
	if opened {
		t.Fatal("a malformed payload must be rejected before the device is opened")
	}
}

func TestSetKeyboardKeysRejectsOutOfRangeIndex(t *testing.T) {
	svc := NewService(fakeReader(t), nil, quietLogger())
	derr := svc.SetKeyboardKeys("200=ff0000", ":1.1")
	if derr == nil || derr.Name != ErrNameBadRequest {
		t.Fatalf("got %v", derr)
	}
}

func TestSetKeyboardKeysRejectsRepeatedIndex(t *testing.T) {
	svc := NewService(fakeReader(t), nil, quietLogger())
	derr := svc.SetKeyboardKeys("0=ff0000,0=00ff00", ":1.1")
	if derr == nil || derr.Name != ErrNameBadRequest {
		t.Fatalf("got %v", derr)
	}
}

func TestSetKeyboardAllAppliesFullRangeInOneTransaction(t *testing.T) {
	svc := NewService(fakeReader(t), nil, quietLogger())
	ft := withFakeKeyboard(t, svc)

	if derr := svc.SetKeyboardAll("00ff88", ":1.1"); derr != nil {
		t.Fatalf("unexpected error: %v", derr)
	}
	frames := ft.frames()
	wantKeys := kbd.DefaultKeyLast - kbd.DefaultKeyFirst + 1
	wantFrames := 1 + (wantKeys+kbd.MaxKeysPerColorSetFrame-1)/kbd.MaxKeysPerColorSetFrame + 1 + 1
	if len(frames) != wantFrames {
		t.Fatalf("got %d frames, want %d for %d keys", len(frames), wantFrames, wantKeys)
	}
}

func TestSetKeyboardAllRejectsBadColor(t *testing.T) {
	svc := NewService(fakeReader(t), nil, quietLogger())
	derr := svc.SetKeyboardAll("not-a-color", ":1.1")
	if derr == nil || derr.Name != ErrNameBadRequest {
		t.Fatalf("got %v", derr)
	}
}

func TestKeyboardOffSendsBlackAcrossTheFullRange(t *testing.T) {
	svc := NewService(fakeReader(t), nil, quietLogger())
	ft := withFakeKeyboard(t, svc)

	if derr := svc.KeyboardOff(":1.1"); derr != nil {
		t.Fatalf("unexpected error: %v", derr)
	}
	frames := ft.frames()
	if len(frames) < 2 {
		t.Fatalf("got %d frames, want at least reset+colorset", len(frames))
	}
	keys := make([]kbd.KeyColor, 0, kbd.MaxKeysPerColorSetFrame)
	for i := 0; i < kbd.MaxKeysPerColorSetFrame; i++ {
		keys = append(keys, kbd.KeyColor{Index: uint8(i)})
	}
	if !bytes.Equal(frames[1], kbd.ColorSetFrame(keys)) {
		t.Errorf("first color set frame is not all black, got % x", frames[1])
	}
}

func TestKeyboardMethodsDeniedErrorNameEndsInDenied(t *testing.T) {
	svc := NewService(fakeReader(t), denyAll{}, quietLogger())
	withFakeKeyboard(t, svc)
	for name, derr := range map[string]*dbus.Error{
		"SetKeyboardKeys": svc.SetKeyboardKeys("0=ff0000", ":1.1"),
		"SetKeyboardAll":  svc.SetKeyboardAll("ff0000", ":1.1"),
		"KeyboardOff":     svc.KeyboardOff(":1.1"),
	} {
		if derr == nil {
			t.Errorf("%s: want a denial", name)
			continue
		}
		if !strings.HasSuffix(derr.Name, ".Denied") {
			t.Errorf("%s: error name %q must end in .Denied", name, derr.Name)
		}
	}
}

func TestKeyboardStatusNeedsNoAuthorization(t *testing.T) {
	svc := NewService(fakeReader(t), denyAll{}, quietLogger())
	withFakeKeyboard(t, svc)
	if _, derr := svc.KeyboardStatus(); derr != nil {
		t.Fatalf("status is read only and must never be denied, got %v", derr)
	}
}

func TestKeyboardOperationsAreSerialized(t *testing.T) {
	svc := NewService(fakeReader(t), nil, quietLogger())
	ft := withFakeKeyboard(t, svc)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			svc.SetKeyboardAll("112233", ":1.1")
		}()
	}
	wg.Wait()

	if ft.closed != 20 {
		t.Fatalf("device closed %d times, want 20 (one full open/close cycle per call)", ft.closed)
	}
}
