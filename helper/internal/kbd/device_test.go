package kbd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/chr0nzz/omarchy-alienware/helper/internal/hidraw"
)

type fakeTransport struct {
	features [][]byte

	featureResponses [][]byte
	featureIdx       int
	getFeatureErr    error

	rawInfo  hidraw.RawInfo
	closed   bool
	closeErr error
}

func (f *fakeTransport) SetFeature(buf []byte) (int, error) {
	f.features = append(f.features, append([]byte(nil), buf...))
	return len(buf), nil
}

func (f *fakeTransport) GetFeature(buf []byte) (int, error) {
	if f.getFeatureErr != nil {
		return 0, f.getFeatureErr
	}
	if f.featureIdx >= len(f.featureResponses) {
		return 0, errors.New("fakeTransport: no queued GetFeature response")
	}
	n := copy(buf, f.featureResponses[f.featureIdx])
	f.featureIdx++
	return n, nil
}

func (f *fakeTransport) RawInfo() (hidraw.RawInfo, error) { return f.rawInfo, nil }
func (f *fakeTransport) Close() error                     { f.closed = true; return f.closeErr }
func (f *fakeTransport) Path() string                     { return "/dev/fake-hidraw" }

func statusFrame(status uint8) []byte {
	b := make([]byte, ReportLength)
	b[0] = 0xcc
	b[2] = status
	return b
}

func TestDeviceResetSendsResetFrame(t *testing.T) {
	ft := &fakeTransport{}
	d := NewWithTransport(ft)

	if err := d.Reset(); err != nil {
		t.Fatalf("Reset() error = %v", err)
	}
	if len(ft.features) != 1 {
		t.Fatalf("got %d feature writes, want 1", len(ft.features))
	}
	if !bytes.Equal(ft.features[0], ResetFrame()) {
		t.Errorf("frame = % x, want reset frame", ft.features[0])
	}
}

func TestDeviceStatusSendsQueryThenReadsFeature(t *testing.T) {
	ft := &fakeTransport{featureResponses: [][]byte{statusFrame(StatusWaitUpdate)}}
	d := NewWithTransport(ft)

	st, err := d.Status()
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if len(ft.features) != 1 {
		t.Fatalf("got %d feature writes, want 1 (the status query)", len(ft.features))
	}
	if !bytes.Equal(ft.features[0], StatusFrame()) {
		t.Errorf("query frame = % x, want status frame", ft.features[0])
	}
	if st.Status != StatusWaitUpdate {
		t.Errorf("Status = %#x, want %#x", st.Status, StatusWaitUpdate)
	}
	if st.Name != "wait-update" {
		t.Errorf("Name = %q, want wait-update", st.Name)
	}
	if st.Ready {
		t.Errorf("Ready = true, want false when status is wait-update")
	}
}

func TestDeviceStatusReadyWhenNotWaitUpdate(t *testing.T) {
	ft := &fakeTransport{featureResponses: [][]byte{statusFrame(0)}}
	d := NewWithTransport(ft)

	st, err := d.Status()
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if !st.Ready {
		t.Errorf("Ready = false, want true when status is zero")
	}
}

func TestSetKeyColorsFullSequence(t *testing.T) {
	ft := &fakeTransport{}
	d := NewWithTransport(ft)

	if err := d.SetKeyColors([]KeyColor{{Index: 4, R: 1, G: 2, B: 3}}); err != nil {
		t.Fatalf("SetKeyColors() error = %v", err)
	}
	if len(ft.features) != 4 {
		t.Fatalf("got %d feature writes, want 4 (reset, color set, loop, update)", len(ft.features))
	}
	if !bytes.Equal(ft.features[0], ResetFrame()) {
		t.Errorf("frame 0 = % x, want reset", ft.features[0])
	}
	if !bytes.Equal(ft.features[1], ColorSetFrame([]KeyColor{{Index: 4, R: 1, G: 2, B: 3}})) {
		t.Errorf("frame 1 = % x, want color set for key 4", ft.features[1])
	}
	if !bytes.Equal(ft.features[2], LoopFrame()) {
		t.Errorf("frame 2 = % x, want loop", ft.features[2])
	}
	if !bytes.Equal(ft.features[3], UpdateFrame()) {
		t.Errorf("frame 3 = % x, want update", ft.features[3])
	}
}

func TestSetKeyColorsChunksAtFifteenPerFrame(t *testing.T) {
	ft := &fakeTransport{}
	d := NewWithTransport(ft)

	keys := make([]KeyColor, MaxKeysPerColorSetFrame+5)
	for i := range keys {
		keys[i] = KeyColor{Index: uint8(i), R: 0xaa}
	}
	if err := d.SetKeyColors(keys); err != nil {
		t.Fatalf("SetKeyColors() error = %v", err)
	}
	wantColorSetFrames := 2
	wantTotal := 1 + wantColorSetFrames + 1 + 1
	if len(ft.features) != wantTotal {
		t.Fatalf("got %d feature writes, want %d (reset, %d color set, loop, update)", len(ft.features), wantTotal, wantColorSetFrames)
	}
	if !bytes.Equal(ft.features[1], ColorSetFrame(keys[:MaxKeysPerColorSetFrame])) {
		t.Errorf("first color set frame did not match the first chunk of keys")
	}
	if !bytes.Equal(ft.features[2], ColorSetFrame(keys[MaxKeysPerColorSetFrame:])) {
		t.Errorf("second color set frame did not match the remaining chunk of keys")
	}
	if !bytes.Equal(ft.features[3], LoopFrame()) {
		t.Errorf("loop frame missing after chunked color set")
	}
	if !bytes.Equal(ft.features[4], UpdateFrame()) {
		t.Errorf("update frame missing after loop")
	}
}

func TestSetAllKeysBuildsFullRange(t *testing.T) {
	ft := &fakeTransport{}
	d := NewWithTransport(ft)

	if err := d.SetAllKeys(1, 2, 3, DefaultKeyFirst, DefaultKeyLast); err != nil {
		t.Fatalf("SetAllKeys() error = %v", err)
	}
	wantKeys := DefaultKeyLast - DefaultKeyFirst + 1
	wantColorSetFrames := (wantKeys + MaxKeysPerColorSetFrame - 1) / MaxKeysPerColorSetFrame
	wantTotal := 1 + wantColorSetFrames + 1 + 1
	if len(ft.features) != wantTotal {
		t.Fatalf("got %d feature writes, want %d for %d keys", len(ft.features), wantTotal, wantKeys)
	}
}

func TestSetBrightnessSendsResetThenTurnOn(t *testing.T) {
	ft := &fakeTransport{}
	d := NewWithTransport(ft)

	if err := d.SetBrightness(0x50); err != nil {
		t.Fatalf("SetBrightness() error = %v", err)
	}
	if len(ft.features) != 2 {
		t.Fatalf("got %d feature writes, want 2", len(ft.features))
	}
	if !bytes.Equal(ft.features[1], TurnOnFrame(0x50)) {
		t.Errorf("frame 1 = % x, want turn-on frame at brightness 0x50", ft.features[1])
	}
}

func TestStatusPropagatesGetFeatureError(t *testing.T) {
	ft := &fakeTransport{getFeatureErr: errors.New("boom")}
	d := NewWithTransport(ft)

	if _, err := d.Status(); err == nil {
		t.Fatalf("Status() should have failed when GetFeature errors")
	}
}

func TestOpenMissingDeviceReturnsTypedError(t *testing.T) {
	_, err := Open("/dev/does-not-exist-alienwarectl-test")
	if err == nil {
		t.Fatalf("expected an error opening a missing device")
	}
	kerr, ok := err.(*Error)
	if !ok {
		t.Fatalf("error %v is not a *kbd.Error", err)
	}
	if kerr.Code() != CodeNoDevice {
		t.Errorf("code = %q, want %q", kerr.Code(), CodeNoDevice)
	}
}

func TestSendFeatureFailureReturnsTypedError(t *testing.T) {
	ft := &fakeTransport{}
	d := NewWithTransport(ft)
	d.t = &failingSetFeatureTransport{fakeTransport: ft}

	err := d.Reset()
	if err == nil {
		t.Fatalf("expected an error when SetFeature fails")
	}
	kerr, ok := err.(*Error)
	if !ok {
		t.Fatalf("error %v is not a *kbd.Error", err)
	}
	if kerr.Code() != CodeInternal {
		t.Errorf("code = %q, want %q", kerr.Code(), CodeInternal)
	}
}

type failingSetFeatureTransport struct {
	*fakeTransport
}

func (f *failingSetFeatureTransport) SetFeature(buf []byte) (int, error) {
	return 0, errors.New("write refused")
}

func TestOpenLockedNodeReportsBusyNotMissing(t *testing.T) {
	path := holdNodeLock(t)
	_, err := Open(path)
	if err == nil {
		t.Fatalf("expected an error opening a node another process holds")
	}
	kerr, ok := err.(*Error)
	if !ok {
		t.Fatalf("error %v is not a *kbd.Error", err)
	}
	if kerr.Code() != CodeBusy {
		t.Errorf("code = %q, want %q", kerr.Code(), CodeBusy)
	}
}

func holdNodeLock(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "node")
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	t.Cleanup(func() { f.Close() })
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatalf("flock %s: %v", path, err)
	}
	return path
}
