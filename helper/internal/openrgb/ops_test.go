package openrgb

import (
	"errors"
	"io"
	"net"
	"reflect"
	"sync"
	"testing"
	"time"
)

type fakeORGBServer struct {
	t    *testing.T
	ln   net.Listener
	mu   sync.Mutex
	ctrl Controller

	updateLEDsCalls     [][]uint32
	updateZoneLEDsCalls []zoneLEDsCall
	updateModeCalls     []modeCall
}

type zoneLEDsCall struct {
	zone   uint32
	colors []uint32
}

type modeCall struct {
	index int32
	mode  Mode
}

func newFakeORGBServer(t *testing.T, ctrl Controller) *fakeORGBServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	s := &fakeORGBServer{t: t, ln: ln, ctrl: ctrl}
	go s.acceptLoop()
	t.Cleanup(func() { ln.Close() })
	return s
}

func (s *fakeORGBServer) addr() string { return s.ln.Addr().String() }

func (s *fakeORGBServer) acceptLoop() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		go s.serve(conn)
	}
}

func (s *fakeORGBServer) serve(conn net.Conn) {
	defer conn.Close()
	for {
		raw := make([]byte, HeaderSize)
		if _, err := io.ReadFull(conn, raw); err != nil {
			return
		}
		h, err := DecodeHeader(raw)
		if err != nil {
			return
		}
		body := make([]byte, h.Size)
		if h.Size > 0 {
			if _, err := io.ReadFull(conn, body); err != nil {
				return
			}
		}
		switch h.PacketID {
		case PktRequestProtocolVersion:
			reply := binaryU32(ClientProtocolVersion)
			frame := append(EncodeHeader(Header{PacketID: PktRequestProtocolVersion, Size: uint32(len(reply))}), reply...)
			conn.Write(frame)
		case PktSetClientName:
		case PktRequestControllerCount:
			reply := binaryU32(1)
			frame := append(EncodeHeader(Header{PacketID: PktRequestControllerCount, Size: uint32(len(reply))}), reply...)
			conn.Write(frame)
		case PktRequestControllerData:
			s.mu.Lock()
			payload := EncodeControllerData(s.ctrl, ClientProtocolVersion)
			s.mu.Unlock()
			frame := append(EncodeHeader(Header{DeviceID: h.DeviceID, PacketID: PktRequestControllerData, Size: uint32(len(payload))}), payload...)
			conn.Write(frame)
		case PktUpdateLEDs:
			colors, err := decodeUpdateLEDsBody(body)
			if err != nil {
				s.t.Errorf("fake server: bad UPDATELEDS body: %v", err)
				continue
			}
			s.mu.Lock()
			s.updateLEDsCalls = append(s.updateLEDsCalls, colors)
			s.mu.Unlock()
		case PktUpdateZoneLEDs:
			zone, colors, err := DecodeUpdateZoneLEDs(body)
			if err != nil {
				s.t.Errorf("fake server: bad UPDATEZONELEDS body: %v", err)
				continue
			}
			s.mu.Lock()
			s.updateZoneLEDsCalls = append(s.updateZoneLEDsCalls, zoneLEDsCall{zone: zone, colors: colors})
			s.mu.Unlock()
		case PktUpdateMode:
			index, mode, err := DecodeUpdateMode(body, ClientProtocolVersion)
			if err != nil {
				s.t.Errorf("fake server: bad UPDATEMODE body: %v", err)
				continue
			}
			s.mu.Lock()
			s.updateModeCalls = append(s.updateModeCalls, modeCall{index: index, mode: mode})
			s.mu.Unlock()
		}
	}
}

func (s *fakeORGBServer) setCtrl(ctrl Controller) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ctrl = ctrl
}

func (s *fakeORGBServer) updateLEDsCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.updateLEDsCalls)
}

func (s *fakeORGBServer) lastUpdateLEDs() []uint32 {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.updateLEDsCalls) == 0 {
		return nil
	}
	return s.updateLEDsCalls[len(s.updateLEDsCalls)-1]
}

func (s *fakeORGBServer) updateModeCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.updateModeCalls)
}

func decodeUpdateLEDsBody(data []byte) ([]uint32, error) {
	c := &cursor{data: data}
	size := c.u32()
	if c.err == nil && int(size) != len(data) {
		return nil, ErrMalformed
	}
	n := int(c.u16())
	if c.err != nil {
		return nil, c.err
	}
	colors := make([]uint32, 0, n)
	for i := 0; i < n; i++ {
		colors = append(colors, c.u32())
	}
	if c.err != nil {
		return nil, c.err
	}
	return colors, nil
}

func waitForCount(t *testing.T, get func() int, want int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if get() == want {
			time.Sleep(75 * time.Millisecond)
			if got := get(); got != want {
				t.Fatalf("count kept changing, settled at %d after reaching %d", got, want)
			}
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for count %d, got %d", want, get())
}

func testController() Controller {
	zones := []Zone{
		{Name: "Zone 0", LEDsMin: 1, LEDsMax: 1, LEDsCount: 1},
		{Name: "Zone 1", LEDsMin: 1, LEDsMax: 1, LEDsCount: 1},
		{Name: "Zone 2", LEDsMin: 1, LEDsMax: 1, LEDsCount: 1},
	}
	leds := []LED{{Name: "LED 0"}, {Name: "LED 1"}, {Name: "LED 2"}}
	modes := []Mode{
		{Name: "Static", Flags: ModeFlagHasPerLEDColor, ColorsMax: 3},
		{Name: "Breathing", Flags: ModeFlagHasSpeed},
	}
	return Controller{
		Name:       "Dell G Series LED Controller",
		ActiveMode: 0,
		Modes:      modes,
		Zones:      zones,
		LEDs:       leds,
		Colors:     []uint32{0, 0, 0},
	}
}

func zeroZoneController() Controller {
	ctrl := testController()
	ctrl.Zones = nil
	ctrl.LEDs = nil
	ctrl.Colors = nil
	return ctrl
}

func openFakeSession(t *testing.T, ctrl Controller) (*Session, *fakeORGBServer) {
	t.Helper()
	server := newFakeORGBServer(t, ctrl)
	session, err := Open(server.addr())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { session.Close() })
	return session, server
}

func TestSetMapCorrectLength(t *testing.T) {
	session, server := openFakeSession(t, testController())
	colors := []uint32{Color(10, 20, 30), Color(40, 50, 60), Color(70, 80, 90)}
	if err := session.SetMap(colors); err != nil {
		t.Fatalf("SetMap: %v", err)
	}
	waitForCount(t, server.updateLEDsCount, 1, time.Second)
	if got := server.lastUpdateLEDs(); !reflect.DeepEqual(got, colors) {
		t.Fatalf("colours sent: got %v, want %v", got, colors)
	}
}

func TestSetMapRejectsWrongLength(t *testing.T) {
	session, server := openFakeSession(t, testController())
	colors := []uint32{Color(10, 20, 30), Color(40, 50, 60)}
	err := session.SetMap(colors)
	if err == nil {
		t.Fatal("want an error for a wrong length list")
	}
	var oerr *Error
	if !errors.As(err, &oerr) || oerr.Code() != CodeBadRequest {
		t.Fatalf("got %v, want a %q error", err, CodeBadRequest)
	}
	if got := server.updateLEDsCount(); got != 0 {
		t.Fatalf("a rejected set-map must not touch the wire, got %d UPDATELEDS calls", got)
	}
}

func TestSetMapIssuesExactlyOneUpdateLEDsCall(t *testing.T) {
	ctrl := testController()
	ctrl.ActiveMode = 1
	session, server := openFakeSession(t, ctrl)
	colors := []uint32{Color(1, 1, 1), Color(2, 2, 2), Color(3, 3, 3)}
	if err := session.SetMap(colors); err != nil {
		t.Fatalf("SetMap: %v", err)
	}
	waitForCount(t, server.updateModeCount, 1, time.Second)
	waitForCount(t, server.updateLEDsCount, 1, time.Second)
	if got := server.lastUpdateLEDs(); !reflect.DeepEqual(got, colors) {
		t.Fatalf("colours sent: got %v, want %v", got, colors)
	}
	if session.ActiveModeName() != "Static" {
		t.Fatalf("active mode after switching for set-map: got %q, want Static", session.ActiveModeName())
	}
}

func TestActiveModeNameReportsSwitchedMode(t *testing.T) {
	ctrl := testController()
	ctrl.ActiveMode = 1
	session, server := openFakeSession(t, ctrl)
	if session.ActiveModeName() != "Breathing" {
		t.Fatalf("before any per-LED write: got %q, want Breathing", session.ActiveModeName())
	}
	if err := session.SetAll(Color(9, 9, 9)); err != nil {
		t.Fatalf("SetAll: %v", err)
	}
	waitForCount(t, server.updateModeCount, 1, time.Second)
	if session.ActiveModeName() != "Static" {
		t.Fatalf("after set-all forced a mode switch: got %q, want Static", session.ActiveModeName())
	}
}

func TestIdentifyRestoresPriorMode(t *testing.T) {
	ctrl := testController()
	ctrl.ActiveMode = 1
	ctrl.Colors = []uint32{Color(5, 5, 5), Color(6, 6, 6), Color(7, 7, 7)}
	session, server := openFakeSession(t, ctrl)
	if err := session.Identify(0, 1, time.Millisecond); err != nil {
		t.Fatalf("Identify: %v", err)
	}
	waitForCount(t, server.updateModeCount, 2, time.Second)
	if session.ActiveModeName() != "Breathing" {
		t.Fatalf("active mode after identify: got %q, want Breathing restored", session.ActiveModeName())
	}
	if session.Ctrl.ActiveMode != 1 {
		t.Fatalf("Ctrl.ActiveMode: got %d, want 1", session.Ctrl.ActiveMode)
	}
}

func TestIdentifyKeepsModeWhenAlreadyPerLED(t *testing.T) {
	ctrl := testController()
	ctrl.Colors = []uint32{Color(5, 5, 5), Color(6, 6, 6), Color(7, 7, 7)}
	session, server := openFakeSession(t, ctrl)
	if err := session.Identify(0, 1, time.Millisecond); err != nil {
		t.Fatalf("Identify: %v", err)
	}
	waitForCount(t, server.updateLEDsCount, 1, time.Second)
	time.Sleep(75 * time.Millisecond)
	if got := server.updateModeCount(); got != 0 {
		t.Fatalf("no mode switch was needed, want 0 UPDATEMODE calls, got %d", got)
	}
	if session.ActiveModeName() != "Static" {
		t.Fatalf("active mode: got %q, want Static", session.ActiveModeName())
	}
}
