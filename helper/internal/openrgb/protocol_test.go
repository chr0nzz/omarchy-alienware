package openrgb

import (
	"bytes"
	"encoding/binary"
	"errors"
	"reflect"
	"testing"
)

func sampleMode() Mode {
	return Mode{
		Name:          "Static",
		Value:         1,
		Flags:         ModeFlagHasBrightness | ModeFlagHasPerLEDColor,
		SpeedMin:      0,
		SpeedMax:      5,
		BrightnessMin: 0,
		BrightnessMax: 100,
		ColorsMin:     1,
		ColorsMax:     1,
		Speed:         2,
		Brightness:    75,
		Direction:     0,
		ColorMode:     1,
		Colors:        []uint32{Color(255, 0, 0)},
	}
}

func sampleController() Controller {
	zones := make([]Zone, 0, 3)
	for i := 0; i < 3; i++ {
		zones = append(zones, Zone{
			Name:      "Unknown",
			Type:      0,
			LEDsMin:   1,
			LEDsMax:   1,
			LEDsCount: 1,
			Flags:     0,
		})
	}
	leds := []LED{{Name: "LED 1", Value: 0}, {Name: "LED 2", Value: 1}, {Name: "LED 3", Value: 2}}
	return Controller{
		Type:        19,
		Name:        "Dell G Series LED Controller",
		Vendor:      "Dell",
		Description: "Dell G Series LED Controller",
		Version:     "",
		Serial:      "",
		Location:    "HID: /dev/hidraw0",
		ActiveMode:  0,
		Modes:       []Mode{sampleMode(), {Name: "Breathing", Value: 2, Flags: ModeFlagHasSpeed}},
		Zones:       zones,
		LEDs:        leds,
		Colors:      []uint32{Color(1, 2, 3), Color(4, 5, 6), Color(7, 8, 9)},
		Flags:       1,
	}
}

func TestHeaderRoundTrip(t *testing.T) {
	in := Header{DeviceID: 7, PacketID: PktUpdateZoneLEDs, Size: 42}
	raw := EncodeHeader(in)
	if len(raw) != HeaderSize {
		t.Fatalf("header is %d bytes, want %d", len(raw), HeaderSize)
	}
	if string(raw[0:4]) != "ORGB" {
		t.Fatalf("magic: got %q", string(raw[0:4]))
	}
	if binary.LittleEndian.Uint32(raw[8:12]) != 1051 {
		t.Fatalf("UPDATEZONELEDS packet id must be 1051 on the wire")
	}
	out, err := DecodeHeader(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != in {
		t.Fatalf("round trip: got %+v, want %+v", out, in)
	}
}

func TestPacketIDs(t *testing.T) {
	cases := map[string]uint32{
		"REQUEST_CONTROLLER_COUNT":      PktRequestControllerCount,
		"REQUEST_CONTROLLER_DATA":       PktRequestControllerData,
		"REQUEST_PROTOCOL_VERSION":      PktRequestProtocolVersion,
		"SET_CLIENT_NAME":               PktSetClientName,
		"DEVICE_LIST_UPDATED":           PktDeviceListUpdated,
		"RGBCONTROLLER_RESIZEZONE":      PktResizeZone,
		"RGBCONTROLLER_UPDATELEDS":      PktUpdateLEDs,
		"RGBCONTROLLER_UPDATEZONELEDS":  PktUpdateZoneLEDs,
		"RGBCONTROLLER_UPDATESINGLELED": PktUpdateSingleLED,
		"RGBCONTROLLER_SETCUSTOMMODE":   PktSetCustomMode,
		"RGBCONTROLLER_UPDATEMODE":      PktUpdateMode,
		"RGBCONTROLLER_SAVEMODE":        PktSaveMode,
	}
	want := map[string]uint32{
		"REQUEST_CONTROLLER_COUNT":      0,
		"REQUEST_CONTROLLER_DATA":       1,
		"REQUEST_PROTOCOL_VERSION":      40,
		"SET_CLIENT_NAME":               50,
		"DEVICE_LIST_UPDATED":           100,
		"RGBCONTROLLER_RESIZEZONE":      1000,
		"RGBCONTROLLER_UPDATELEDS":      1050,
		"RGBCONTROLLER_UPDATEZONELEDS":  1051,
		"RGBCONTROLLER_UPDATESINGLELED": 1052,
		"RGBCONTROLLER_SETCUSTOMMODE":   1100,
		"RGBCONTROLLER_UPDATEMODE":      1101,
		"RGBCONTROLLER_SAVEMODE":        1102,
	}
	for name, got := range cases {
		if got != want[name] {
			t.Errorf("%s: got %d, want %d", name, got, want[name])
		}
	}
}

func TestDecodeHeaderRejectsBadMagic(t *testing.T) {
	raw := EncodeHeader(Header{})
	copy(raw[0:4], "NOPE")
	if _, err := DecodeHeader(raw); !errors.Is(err, ErrMalformed) {
		t.Fatalf("got %v, want ErrMalformed", err)
	}
	if _, err := DecodeHeader(raw[:8]); !errors.Is(err, ErrMalformed) {
		t.Fatalf("short header: got %v, want ErrMalformed", err)
	}
}

func TestDecodeHeaderRejectsOversizePayload(t *testing.T) {
	raw := EncodeHeader(Header{Size: MaxPacketSize + 1})
	if _, err := DecodeHeader(raw); !errors.Is(err, ErrMalformed) {
		t.Fatalf("got %v, want ErrMalformed", err)
	}
}

func TestColorWireOrder(t *testing.T) {
	c := Color(0x12, 0x34, 0x56)
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], c)
	if !bytes.Equal(b[:], []byte{0x12, 0x34, 0x56, 0x00}) {
		t.Fatalf("wire bytes: got %v, want [18 52 86 0]", b)
	}
	r, g, bl := ColorParts(c)
	if r != 0x12 || g != 0x34 || bl != 0x56 {
		t.Fatalf("parts: got %d %d %d", r, g, bl)
	}
}

func TestParseHex(t *testing.T) {
	c, err := ParseHex("#FF8800")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r, g, b := ColorParts(c)
	if r != 0xFF || g != 0x88 || b != 0x00 {
		t.Fatalf("got %d %d %d", r, g, b)
	}
	if HexString(c) != "ff8800" {
		t.Fatalf("HexString: got %q", HexString(c))
	}
	for _, bad := range []string{"", "fff", "gggggg", "1234567"} {
		if _, err := ParseHex(bad); err == nil {
			t.Errorf("%q should not parse", bad)
		}
	}
}

func TestControllerDataRoundTripAllVersions(t *testing.T) {
	for _, version := range []uint32{1, 2, 3, 4, 5} {
		in := sampleController()
		if version < 1 {
			in.Vendor = ""
		}
		if version < 3 {
			for i := range in.Modes {
				in.Modes[i].BrightnessMin = 0
				in.Modes[i].BrightnessMax = 0
				in.Modes[i].Brightness = 0
			}
		}
		if version < 5 {
			in.Flags = 0
		}
		raw := EncodeControllerData(in, version)
		if int(binary.LittleEndian.Uint32(raw[:4])) != len(raw) {
			t.Fatalf("version %d: leading data_size must equal the blob length", version)
		}
		out, err := DecodeControllerData(raw, version)
		if err != nil {
			t.Fatalf("version %d: %v", version, err)
		}
		out.Index = in.Index
		if !reflect.DeepEqual(in, out) {
			t.Fatalf("version %d round trip mismatch:\n in: %+v\nout: %+v", version, in, out)
		}
	}
}

func TestControllerDataVendorOnlyAtVersion1(t *testing.T) {
	in := sampleController()
	in.Flags = 0
	raw := EncodeControllerData(in, 0)
	out, err := DecodeControllerData(raw, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Vendor != "" {
		t.Fatalf("protocol 0 has no vendor field, got %q", out.Vendor)
	}
	if out.Name != in.Name {
		t.Fatalf("name: got %q, want %q", out.Name, in.Name)
	}
}

func TestControllerDataRejectsSizeMismatch(t *testing.T) {
	raw := EncodeControllerData(sampleController(), 4)
	binary.LittleEndian.PutUint32(raw[:4], uint32(len(raw)+8))
	if _, err := DecodeControllerData(raw, 4); !errors.Is(err, ErrMalformed) {
		t.Fatalf("got %v, want ErrMalformed", err)
	}
}

func TestControllerDataRejectsTruncated(t *testing.T) {
	raw := EncodeControllerData(sampleController(), 4)
	short := raw[:len(raw)/2]
	binary.LittleEndian.PutUint32(short[:4], uint32(len(short)))
	if _, err := DecodeControllerData(short, 4); !errors.Is(err, ErrMalformed) {
		t.Fatalf("got %v, want ErrMalformed", err)
	}
}

func TestControllerDataRejectsEmpty(t *testing.T) {
	if _, err := DecodeControllerData(nil, 4); !errors.Is(err, ErrMalformed) {
		t.Fatalf("got %v, want ErrMalformed", err)
	}
}

func TestUpdateZoneLEDsRoundTrip(t *testing.T) {
	colors := []uint32{Color(255, 0, 0), Color(0, 255, 0), Color(0, 0, 255)}
	raw := EncodeUpdateZoneLEDs(4, colors)
	if int(binary.LittleEndian.Uint32(raw[:4])) != len(raw) {
		t.Fatal("data_size must equal the payload length")
	}
	if len(raw) != 4+4+2+3*4 {
		t.Fatalf("payload is %d bytes, want %d", len(raw), 4+4+2+12)
	}
	zone, got, err := DecodeUpdateZoneLEDs(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if zone != 4 {
		t.Fatalf("zone: got %d, want 4", zone)
	}
	if !reflect.DeepEqual(got, colors) {
		t.Fatalf("colors: got %v, want %v", got, colors)
	}
}

func TestUpdateLEDsLayout(t *testing.T) {
	raw := EncodeUpdateLEDs([]uint32{Color(1, 2, 3), Color(4, 5, 6)})
	if int(binary.LittleEndian.Uint32(raw[:4])) != len(raw) {
		t.Fatal("data_size must equal the payload length")
	}
	if len(raw) != 4+2+8 {
		t.Fatalf("payload is %d bytes, want %d", len(raw), 4+2+8)
	}
	if binary.LittleEndian.Uint16(raw[4:6]) != 2 {
		t.Fatal("num_colors must be a uint16 directly after data_size")
	}
}

func TestUpdateModeRoundTrip(t *testing.T) {
	m := sampleMode()
	raw := EncodeUpdateMode(3, m, 4)
	if int(binary.LittleEndian.Uint32(raw[:4])) != len(raw) {
		t.Fatal("data_size must equal the payload length")
	}
	index, got, err := DecodeUpdateMode(raw, 4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if index != 3 {
		t.Fatalf("mode index: got %d, want 3", index)
	}
	if !reflect.DeepEqual(got, m) {
		t.Fatalf("mode round trip:\n in: %+v\nout: %+v", m, got)
	}
}

func TestUpdateModeDropsBrightnessBelowVersion3(t *testing.T) {
	m := sampleMode()
	raw := EncodeUpdateMode(0, m, 2)
	_, got, err := DecodeUpdateMode(raw, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Brightness != 0 || got.BrightnessMax != 0 {
		t.Fatalf("protocol 2 carries no brightness, got %+v", got)
	}
	if got.Speed != m.Speed || got.ColorMode != m.ColorMode {
		t.Fatalf("the remaining fields must still line up: %+v", got)
	}
}

func TestUpdateModeRejectsSizeMismatch(t *testing.T) {
	raw := EncodeUpdateMode(0, sampleMode(), 4)
	binary.LittleEndian.PutUint32(raw[:4], 9999)
	if _, _, err := DecodeUpdateMode(raw, 4); !errors.Is(err, ErrMalformed) {
		t.Fatalf("got %v, want ErrMalformed", err)
	}
}

func TestStringEncodingIncludesNull(t *testing.T) {
	b := &buf{}
	b.str("ab")
	if !bytes.Equal(b.data, []byte{3, 0, 'a', 'b', 0}) {
		t.Fatalf("got %v, want [3 0 97 98 0]", b.data)
	}
	c := &cursor{data: b.data}
	if got := c.str(); got != "ab" {
		t.Fatalf("decoded %q", got)
	}
}

func TestScaleBrightness(t *testing.T) {
	m := Mode{BrightnessMin: 0, BrightnessMax: 100}
	if got := ScaleBrightness(m, 50); got != 50 {
		t.Errorf("50 percent of 0-100: got %d", got)
	}
	m = Mode{BrightnessMin: 10, BrightnessMax: 20}
	if got := ScaleBrightness(m, 0); got != 10 {
		t.Errorf("0 percent: got %d", got)
	}
	if got := ScaleBrightness(m, 100); got != 20 {
		t.Errorf("100 percent: got %d", got)
	}
	if got := ScaleBrightness(m, 55); got != 16 {
		t.Errorf("55 percent: got %d, want 16", got)
	}
	flat := Mode{BrightnessMin: 5, BrightnessMax: 5}
	if got := ScaleBrightness(flat, 70); got != 5 {
		t.Errorf("flat range: got %d, want 5", got)
	}
}

func TestResolveZone(t *testing.T) {
	ctrl := sampleController()
	ctrl.Zones[0].Name = "Keyboard left"
	ctrl.Zones[1].Name = "Keyboard right"
	ctrl.Zones[2].Name = "Logo"
	if i, err := ResolveZone(ctrl, "2"); err != nil || i != 2 {
		t.Errorf("numeric: got %d, %v", i, err)
	}
	if i, err := ResolveZone(ctrl, "logo"); err != nil || i != 2 {
		t.Errorf("exact name: got %d, %v", i, err)
	}
	if i, err := ResolveZone(ctrl, "left"); err != nil || i != 0 {
		t.Errorf("substring: got %d, %v", i, err)
	}
	if _, err := ResolveZone(ctrl, "keyboard"); err == nil {
		t.Error("an ambiguous substring should fail")
	}
	if _, err := ResolveZone(ctrl, "99"); err == nil {
		t.Error("an out of range index should fail")
	}
	if _, err := ResolveZone(ctrl, ""); err == nil {
		t.Error("an empty zone should fail")
	}
}

func TestModeIndex(t *testing.T) {
	ctrl := sampleController()
	if i, err := ModeIndex(ctrl, "breathing"); err != nil || i != 1 {
		t.Errorf("got %d, %v", i, err)
	}
	err := func() error { _, e := ModeIndex(ctrl, "Rainbow"); return e }()
	var oerr *Error
	if !errors.As(err, &oerr) || oerr.Code() != CodeNotSupported {
		t.Errorf("unknown mode should map to not-supported, got %v", err)
	}
}

func TestDialRefusedMapsToNoOpenRGB(t *testing.T) {
	_, err := Dial("127.0.0.1:1")
	var oerr *Error
	if !errors.As(err, &oerr) {
		t.Fatalf("got %T %v, want *openrgb.Error", err, err)
	}
	if oerr.Code() != CodeNoOpenRGB {
		t.Fatalf("code: got %q, want %q", oerr.Code(), CodeNoOpenRGB)
	}
}
