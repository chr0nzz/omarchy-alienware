package cli

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
	"testing"

	"github.com/chr0nzz/omarchy-alienware/helper/internal/openrgb"
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

func fakeRGBController() openrgb.Controller {
	zones := []openrgb.Zone{
		{Name: "Zone 0", LEDsMin: 1, LEDsMax: 1, LEDsCount: 1},
		{Name: "Zone 1", LEDsMin: 1, LEDsMax: 1, LEDsCount: 1},
	}
	leds := []openrgb.LED{{Name: "LED 0"}, {Name: "LED 1"}}
	modes := []openrgb.Mode{
		{Name: "Static", Flags: openrgb.ModeFlagHasPerLEDColor, ColorsMax: 2},
		{Name: "Breathing", Flags: openrgb.ModeFlagHasSpeed},
	}
	return openrgb.Controller{
		Name:       "Dell G Series LED Controller",
		ActiveMode: 1,
		Modes:      modes,
		Zones:      zones,
		LEDs:       leds,
		Colors:     []uint32{0, 0},
	}
}

func startFakeRGBServer(t *testing.T, ctrl openrgb.Controller) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go serveFakeRGB(conn, ctrl)
		}
	}()
	return ln.Addr().String()
}

func serveFakeRGB(conn net.Conn, ctrl openrgb.Controller) {
	defer conn.Close()
	for {
		raw := make([]byte, openrgb.HeaderSize)
		if _, err := io.ReadFull(conn, raw); err != nil {
			return
		}
		h, err := openrgb.DecodeHeader(raw)
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
		case openrgb.PktRequestProtocolVersion:
			reply := u32Bytes(openrgb.ClientProtocolVersion)
			conn.Write(append(openrgb.EncodeHeader(openrgb.Header{PacketID: openrgb.PktRequestProtocolVersion, Size: uint32(len(reply))}), reply...))
		case openrgb.PktRequestControllerCount:
			reply := u32Bytes(1)
			conn.Write(append(openrgb.EncodeHeader(openrgb.Header{PacketID: openrgb.PktRequestControllerCount, Size: uint32(len(reply))}), reply...))
		case openrgb.PktRequestControllerData:
			payload := openrgb.EncodeControllerData(ctrl, openrgb.ClientProtocolVersion)
			conn.Write(append(openrgb.EncodeHeader(openrgb.Header{DeviceID: h.DeviceID, PacketID: openrgb.PktRequestControllerData, Size: uint32(len(payload))}), payload...))
		}
	}
}

func u32Bytes(v uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	return b
}

func TestRGBSetIncludesActiveMode(t *testing.T) {
	addr := startFakeRGBServer(t, fakeRGBController())
	t.Setenv(openrgb.AddrEnv, addr)
	code, parsed, raw := run(t, "", "rgb", "set", "0", "ff0000")
	if code != 0 {
		t.Fatalf("exit code: got %d, raw=%s", code, raw)
	}
	if parsed["activeMode"] != "Static" {
		t.Fatalf("activeMode: got %v, want Static, raw=%s", parsed["activeMode"], raw)
	}
}

func TestRGBSetAllIncludesActiveMode(t *testing.T) {
	addr := startFakeRGBServer(t, fakeRGBController())
	t.Setenv(openrgb.AddrEnv, addr)
	code, parsed, raw := run(t, "", "rgb", "set-all", "00ff00")
	if code != 0 {
		t.Fatalf("exit code: got %d, raw=%s", code, raw)
	}
	if parsed["activeMode"] != "Static" {
		t.Fatalf("activeMode: got %v, want Static, raw=%s", parsed["activeMode"], raw)
	}
}

func TestRGBSetMapIncludesActiveModeAndColors(t *testing.T) {
	addr := startFakeRGBServer(t, fakeRGBController())
	t.Setenv(openrgb.AddrEnv, addr)
	code, parsed, raw := run(t, "", "rgb", "set-map", "ff0000,00ff00")
	if code != 0 {
		t.Fatalf("exit code: got %d, raw=%s", code, raw)
	}
	if parsed["activeMode"] != "Static" {
		t.Fatalf("activeMode: got %v, want Static, raw=%s", parsed["activeMode"], raw)
	}
	colors, ok := parsed["colors"].([]any)
	if !ok || len(colors) != 2 || colors[0] != "ff0000" || colors[1] != "00ff00" {
		t.Fatalf("colors: got %v, raw=%s", parsed["colors"], raw)
	}
}

func TestRGBSetMapRejectsWrongLength(t *testing.T) {
	addr := startFakeRGBServer(t, fakeRGBController())
	t.Setenv(openrgb.AddrEnv, addr)
	code, parsed, raw := run(t, "", "rgb", "set-map", "ff0000")
	if code == 0 {
		t.Fatalf("want a non zero exit code, raw=%s", raw)
	}
	if parsed["ok"] != false || parsed["code"] != "bad-request" {
		t.Fatalf("got %v, raw=%s", parsed, raw)
	}
}

func TestRGBIdentifyIncludesActiveMode(t *testing.T) {
	addr := startFakeRGBServer(t, fakeRGBController())
	t.Setenv(openrgb.AddrEnv, addr)
	code, parsed, raw := run(t, "", "rgb", "identify", "0")
	if code != 0 {
		t.Fatalf("exit code: got %d, raw=%s", code, raw)
	}
	if parsed["activeMode"] != "Breathing" {
		t.Fatalf("activeMode: got %v, want Breathing restored, raw=%s", parsed["activeMode"], raw)
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
	fake := &fakeCLIResetter{err: errors.New("no AW-ELC controller (187c:0550) found on the USB bus")}
	withFakeResetter(t, fake)
	code, parsed, raw := run(t, "", "rgb", "reset")
	if code == 0 {
		t.Fatalf("want a non zero exit code, raw=%s", raw)
	}
	if parsed["ok"] != false {
		t.Fatalf("got %v, raw=%s", parsed, raw)
	}
}

func TestRGBResetSkipsOpeningTheOpenRGBSession(t *testing.T) {
	fake := &fakeCLIResetter{node: "/dev/bus/usb/003/017"}
	withFakeResetter(t, fake)
	t.Setenv(openrgb.AddrEnv, "127.0.0.1:1")
	code, _, raw := run(t, "", "rgb", "reset")
	if code != 0 {
		t.Fatalf("rgb reset must not require an OpenRGB connection, exit code %d, raw=%s", code, raw)
	}
}

func TestRGBStatusShapeUnchanged(t *testing.T) {
	addr := startFakeRGBServer(t, fakeRGBController())
	t.Setenv(openrgb.AddrEnv, addr)
	code, parsed, raw := run(t, "", "rgb", "status")
	if code != 0 {
		t.Fatalf("exit code: got %d, raw=%s", code, raw)
	}
	if _, present := parsed["activeMode"]; present {
		t.Fatalf("rgb status must not gain a top level activeMode field, raw=%s", raw)
	}
	device, ok := parsed["device"].(map[string]any)
	if !ok {
		t.Fatalf("device object missing, raw=%s", raw)
	}
	if device["activeMode"] != "Breathing" {
		t.Fatalf("device.activeMode: got %v, raw=%s", device["activeMode"], raw)
	}
}
