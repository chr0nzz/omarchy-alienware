package elc

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/chr0nzz/omarchy-alienware/helper/internal/hidraw"
)

type fakeTransport struct {
	outputs  [][]byte
	writes   [][]byte
	features [][]byte

	inputResponses  [][]byte
	inputIdx        int
	inputRepeatLast bool

	featureResponses [][]byte
	featureIdx       int

	rawInfo  hidraw.RawInfo
	closed   bool
	closeErr error
}

func (f *fakeTransport) SetOutputReport(buf []byte) (int, error) {
	f.outputs = append(f.outputs, append([]byte(nil), buf...))
	return len(buf), nil
}

func (f *fakeTransport) Write(buf []byte) (int, error) {
	f.writes = append(f.writes, append([]byte(nil), buf...))
	return len(buf), nil
}

func (f *fakeTransport) GetInput(buf []byte) (int, error) {
	if f.inputIdx >= len(f.inputResponses) {
		if f.inputRepeatLast && len(f.inputResponses) > 0 {
			return copy(buf, f.inputResponses[len(f.inputResponses)-1]), nil
		}
		return 0, errors.New("fakeTransport: no queued GetInput response")
	}
	n := copy(buf, f.inputResponses[f.inputIdx])
	f.inputIdx++
	return n, nil
}

func (f *fakeTransport) SetFeature(buf []byte) (int, error) {
	f.features = append(f.features, append([]byte(nil), buf...))
	return len(buf), nil
}

func (f *fakeTransport) GetFeature(buf []byte) (int, error) {
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
	b[2] = status
	return b
}

func TestResetSendsRemoveThenStartNew(t *testing.T) {
	ft := &fakeTransport{inputResponses: [][]byte{statusFrame(StatusV4Ready)}}
	d := NewWithTransport(ft, WriteModeOutput)

	if err := d.Reset(); err != nil {
		t.Fatalf("Reset() error = %v", err)
	}
	if len(ft.outputs) != 2 {
		t.Fatalf("got %d output frames, want 2", len(ft.outputs))
	}
	if !bytes.Equal(ft.outputs[0], ControlFrame(ControlRemove, ControlIDCommon)) {
		t.Errorf("first frame = % x, want remove control frame", ft.outputs[0])
	}
	if !bytes.Equal(ft.outputs[1], ControlFrame(ControlStartNew, ControlIDCommon)) {
		t.Errorf("second frame = % x, want start-new control frame", ft.outputs[1])
	}
}

func TestApplySendsFinishPlay(t *testing.T) {
	ft := &fakeTransport{}
	d := NewWithTransport(ft, WriteModeOutput)

	if err := d.Apply(); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if len(ft.outputs) != 1 {
		t.Fatalf("got %d output frames, want 1", len(ft.outputs))
	}
	if !bytes.Equal(ft.outputs[0], ControlFrame(ControlFinishPlay, ControlIDCommon)) {
		t.Errorf("frame = % x, want finish-play control frame", ft.outputs[0])
	}
}

func TestSetColorZonesChunksAtMaxPerFrame(t *testing.T) {
	ft := &fakeTransport{}
	d := NewWithTransport(ft, WriteModeOutput)

	zones := make([]uint8, 30)
	for i := range zones {
		zones[i] = uint8(i)
	}
	if err := d.SetColorZones(1, 2, 3, zones); err != nil {
		t.Fatalf("SetColorZones() error = %v", err)
	}
	if len(ft.outputs) != 2 {
		t.Fatalf("got %d frames, want 2 for %d zones with %d per frame", len(ft.outputs), len(zones), MaxColorZonesPerFrame)
	}
	if !bytes.Equal(ft.outputs[0], SetOneColorFrame(1, 2, 3, zones[:MaxColorZonesPerFrame])) {
		t.Errorf("first frame did not match the first chunk of zones")
	}
	if !bytes.Equal(ft.outputs[1], SetOneColorFrame(1, 2, 3, zones[MaxColorZonesPerFrame:])) {
		t.Errorf("second frame did not match the remaining chunk of zones")
	}
}

func TestSetZoneColorFullFlow(t *testing.T) {
	ft := &fakeTransport{inputResponses: [][]byte{statusFrame(StatusV4Ready)}}
	d := NewWithTransport(ft, WriteModeOutput)

	if err := d.SetZoneColor(0xff, 0x00, 0x80, []uint8{4}); err != nil {
		t.Fatalf("SetZoneColor() error = %v", err)
	}
	if len(ft.outputs) != 4 {
		t.Fatalf("got %d frames, want 4 (remove, start-new, color, finish-play), frames: %v", len(ft.outputs), ft.outputs)
	}
	if !bytes.Equal(ft.outputs[2], SetOneColorFrame(0xff, 0x00, 0x80, []uint8{4})) {
		t.Errorf("third frame = % x, want the set-one-color frame for zone 4", ft.outputs[2])
	}
	if !bytes.Equal(ft.outputs[3], ControlFrame(ControlFinishPlay, ControlIDCommon)) {
		t.Errorf("fourth frame = % x, want finish-play", ft.outputs[3])
	}
}

func TestSetColorGroupsSendsOneResetAndOneApplyForMultipleRegions(t *testing.T) {
	ft := &fakeTransport{inputResponses: [][]byte{statusFrame(StatusV4Ready)}}
	d := NewWithTransport(ft, WriteModeOutput)

	groups := []ColorZones{
		{R: 0xff, G: 0, B: 0, ZoneIDs: []uint8{1}},
		{R: 0, G: 0xff, B: 0, ZoneIDs: []uint8{0}},
	}
	if err := d.SetColorGroups(groups); err != nil {
		t.Fatalf("SetColorGroups() error = %v", err)
	}
	if len(ft.outputs) != 5 {
		t.Fatalf("got %d frames, want 5 (remove, start-new, colour, colour, finish-play), frames: %v", len(ft.outputs), ft.outputs)
	}
	if !bytes.Equal(ft.outputs[0], ControlFrame(ControlRemove, ControlIDCommon)) {
		t.Errorf("first frame = % x, want remove control frame", ft.outputs[0])
	}
	if !bytes.Equal(ft.outputs[1], ControlFrame(ControlStartNew, ControlIDCommon)) {
		t.Errorf("second frame = % x, want start-new control frame", ft.outputs[1])
	}
	if !bytes.Equal(ft.outputs[2], SetOneColorFrame(0xff, 0, 0, []uint8{1})) {
		t.Errorf("third frame = % x, want the logo colour frame", ft.outputs[2])
	}
	if !bytes.Equal(ft.outputs[3], SetOneColorFrame(0, 0xff, 0, []uint8{0})) {
		t.Errorf("fourth frame = % x, want the power colour frame", ft.outputs[3])
	}
}

func TestWaitForReadyTimesOutWhenAlwaysBusy(t *testing.T) {
	ft := &fakeTransport{
		inputResponses:  [][]byte{statusFrame(StatusV4Busy)},
		inputRepeatLast: true,
	}
	d := NewWithTransport(ft, WriteModeOutput)

	_, err := d.WaitForReady(20 * time.Millisecond)
	if !errors.Is(err, ErrDeviceTimeout) {
		t.Fatalf("WaitForReady() error = %v, want ErrDeviceTimeout", err)
	}
}

func TestWaitForReadyReturnsOnZeroStatus(t *testing.T) {
	ft := &fakeTransport{inputResponses: [][]byte{statusFrame(0)}}
	d := NewWithTransport(ft, WriteModeOutput)

	st, err := d.WaitForReady(time.Second)
	if err != nil {
		t.Fatalf("WaitForReady() error = %v", err)
	}
	if st.Status != 0 || st.Name != "zero" {
		t.Errorf("status = %+v, want zero/zero", st)
	}
}

func TestStatusParsesStatusByte(t *testing.T) {
	ft := &fakeTransport{inputResponses: [][]byte{statusFrame(StatusV4WaitUpdate)}}
	d := NewWithTransport(ft, WriteModeOutput)

	st, err := d.Status()
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if st.Status != StatusV4WaitUpdate {
		t.Errorf("Status = %d, want %d", st.Status, StatusV4WaitUpdate)
	}
	if st.Name != "wait-update" {
		t.Errorf("Name = %q, want wait-update", st.Name)
	}
}

func TestSendUsesFeatureTransportWhenSelected(t *testing.T) {
	ft := &fakeTransport{}
	d := NewWithTransport(ft, WriteModeFeature)

	if err := d.Apply(); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if len(ft.features) != 1 {
		t.Fatalf("got %d feature writes, want 1", len(ft.features))
	}
	if len(ft.outputs) != 0 {
		t.Fatalf("got %d output writes, want 0 when using the feature transport", len(ft.outputs))
	}
}

func TestSendUsesPlainWriteWhenSelected(t *testing.T) {
	ft := &fakeTransport{}
	d := NewWithTransport(ft, WriteModeWrite)

	if err := d.Apply(); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if len(ft.writes) != 1 {
		t.Fatalf("got %d plain writes, want 1", len(ft.writes))
	}
}

func TestDimChunksAtMaxPerFrame(t *testing.T) {
	ft := &fakeTransport{}
	d := NewWithTransport(ft, WriteModeOutput)

	zones := make([]uint8, MaxTurnOnZonesPerFrame+2)
	for i := range zones {
		zones[i] = uint8(i)
	}
	if err := d.Dim(75, zones); err != nil {
		t.Fatalf("Dim() error = %v", err)
	}
	if len(ft.outputs) != 2 {
		t.Fatalf("got %d frames, want 2", len(ft.outputs))
	}
	if ft.outputs[0][3] != 25 {
		t.Errorf("dim byte = %d, want 25 for 75%% brightness", ft.outputs[0][3])
	}
}

func TestParseWriteMode(t *testing.T) {
	cases := map[string]WriteMode{"": WriteModeOutput, "output": WriteModeOutput, "write": WriteModeWrite, "feature": WriteModeFeature}
	for s, want := range cases {
		got, err := ParseWriteMode(s)
		if err != nil {
			t.Fatalf("ParseWriteMode(%q) error = %v", s, err)
		}
		if got != want {
			t.Errorf("ParseWriteMode(%q) = %v, want %v", s, got, want)
		}
	}
	if _, err := ParseWriteMode("bogus"); err == nil {
		t.Errorf("ParseWriteMode(bogus) should have failed")
	}
}

func TestSelectZonesRejectsOversizedList(t *testing.T) {
	ft := &fakeTransport{}
	d := NewWithTransport(ft, WriteModeOutput)
	zones := make([]uint8, MaxSelectZonesPerFrame+1)
	if err := d.SelectZones(true, zones); err == nil {
		t.Fatalf("SelectZones() should reject a zone list longer than %d", MaxSelectZonesPerFrame)
	}
}

func TestAddActionRejectsTooManyPhases(t *testing.T) {
	ft := &fakeTransport{}
	d := NewWithTransport(ft, WriteModeOutput)
	phases := make([]Phase, MaxActionPhasesPerFrame+1)
	if err := d.AddAction(phases); err == nil {
		t.Fatalf("AddAction() should reject more than %d phases", MaxActionPhasesPerFrame)
	}
}

func TestOpenMissingDeviceReturnsTypedError(t *testing.T) {
	_, err := Open("/dev/does-not-exist-alienwarectl-test", WriteModeOutput)
	if err == nil {
		t.Fatalf("expected an error opening a missing device")
	}
	eerr, ok := err.(*Error)
	if !ok {
		t.Fatalf("error %v is not an *elc.Error", err)
	}
	if eerr.Code() != CodeNoDevice {
		t.Errorf("code = %q, want %q", eerr.Code(), CodeNoDevice)
	}
}

func TestOpenLockedNodeReportsBusyNotMissing(t *testing.T) {
	path := holdNodeLock(t)
	_, err := Open(path, WriteModeOutput)
	if err == nil {
		t.Fatalf("expected an error opening a node another process holds")
	}
	eerr, ok := err.(*Error)
	if !ok {
		t.Fatalf("error %v is not an *elc.Error", err)
	}
	if eerr.Code() != CodeBusy {
		t.Errorf("code = %q, want %q", eerr.Code(), CodeBusy)
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
