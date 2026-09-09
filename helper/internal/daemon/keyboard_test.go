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
	opens    int
	failWith error
}

func (f *fakeKbdTransport) SetFeature(buf []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failWith != nil {
		return 0, f.failWith
	}
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
		ft.mu.Lock()
		ft.opens++
		ft.mu.Unlock()
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
	if ft.closed != 0 {
		t.Fatalf("device closed %d times, want 0, a successful call keeps the cached device open", ft.closed)
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

	if ft.opens != 1 {
		t.Fatalf("device opened %d times, want 1 for 20 serialized calls", ft.opens)
	}
	if ft.closed != 0 {
		t.Fatalf("device closed %d times, want 0 while every call succeeds", ft.closed)
	}
	wantKeys := kbd.DefaultKeyLast - kbd.DefaultKeyFirst + 1
	perCall := 1 + (wantKeys+kbd.MaxKeysPerColorSetFrame-1)/kbd.MaxKeysPerColorSetFrame + 1 + 1
	if len(ft.frames()) != 20*perCall {
		t.Fatalf("got %d frames, want %d for 20 uninterleaved transactions", len(ft.frames()), 20*perCall)
	}
}

func TestApplyKeyboardReusesTheCachedDeviceAcrossCalls(t *testing.T) {
	svc := NewService(fakeReader(t), nil, quietLogger())
	ft := withFakeKeyboard(t, svc)

	var seen []*kbd.Device
	for i := 0; i < 2; i++ {
		if err := svc.applyKeyboard(func(dev *kbd.Device) error {
			seen = append(seen, dev)
			return nil
		}); err != nil {
			t.Fatalf("call %d: unexpected error: %v", i, err)
		}
	}
	if ft.opens != 1 {
		t.Fatalf("device opened %d times, want 1", ft.opens)
	}
	if ft.closed != 0 {
		t.Fatalf("device closed %d times, want 0", ft.closed)
	}
	if seen[0] != seen[1] {
		t.Fatal("the second call got a different device, the cache was not reused")
	}
}

func TestApplyKeyboardReopensOnceAfterAnError(t *testing.T) {
	svc := NewService(fakeReader(t), nil, quietLogger())
	transports := []*fakeKbdTransport{{}, {}}
	opens := 0
	svc.openKeyboard = func() (*kbd.Device, error) {
		if opens >= len(transports) {
			return nil, errors.New("opened more times than the test expects")
		}
		dev := kbd.NewWithTransport(transports[opens])
		opens++
		return dev, nil
	}

	calls := 0
	err := svc.applyKeyboard(func(dev *kbd.Device) error {
		calls++
		if calls == 1 {
			return errors.New("write failed")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("the retry should have succeeded, got %v", err)
	}
	if calls != 2 {
		t.Fatalf("fn ran %d times, want exactly 2", calls)
	}
	if opens != 2 {
		t.Fatalf("device opened %d times, want 2", opens)
	}
	if transports[0].closed != 1 {
		t.Fatalf("the failed device was closed %d times, want 1", transports[0].closed)
	}
	if transports[1].closed != 0 {
		t.Fatalf("the reopened device was closed %d times, want 0", transports[1].closed)
	}
}

func TestApplyKeyboardReturnsTheFirstErrorWhenTheReopenFails(t *testing.T) {
	svc := NewService(fakeReader(t), nil, quietLogger())
	opens := 0
	svc.openKeyboard = func() (*kbd.Device, error) {
		opens++
		if opens > 1 {
			return nil, errors.New("the device went away")
		}
		return kbd.NewWithTransport(&fakeKbdTransport{}), nil
	}

	first := errors.New("write failed")
	err := svc.applyKeyboard(func(dev *kbd.Device) error { return first })
	if !errors.Is(err, first) {
		t.Fatalf("got %v, want the original error from the first attempt", err)
	}
}

func TestCloseKeyboardIsSafeWhenNothingIsCached(t *testing.T) {
	svc := NewService(fakeReader(t), nil, quietLogger())
	ft := withFakeKeyboard(t, svc)

	svc.CloseKeyboard()
	if ft.opens != 0 || ft.closed != 0 {
		t.Fatalf("opens=%d closed=%d, want 0 and 0 for an empty cache", ft.opens, ft.closed)
	}

	if derr := svc.SetKeyboardAll("112233", ":1.1"); derr != nil {
		t.Fatalf("unexpected error: %v", derr)
	}
	svc.CloseKeyboard()
	svc.CloseKeyboard()
	if ft.closed != 1 {
		t.Fatalf("device closed %d times, want exactly 1", ft.closed)
	}
}
