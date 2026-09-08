package openrgb

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
)

const (
	Magic         = "ORGB"
	HeaderSize    = 16
	MaxPacketSize = 8 * 1024 * 1024

	ClientProtocolVersion = uint32(4)
)

const (
	PktRequestControllerCount = uint32(0)
	PktRequestControllerData  = uint32(1)
	PktRequestProtocolVersion = uint32(40)
	PktSetClientName          = uint32(50)
	PktDeviceListUpdated      = uint32(100)
	PktDetectionStarted       = uint32(101)
	PktDetectionProgress      = uint32(102)
	PktDetectionComplete      = uint32(103)
	PktResizeZone             = uint32(1000)
	PktUpdateLEDs             = uint32(1050)
	PktUpdateZoneLEDs         = uint32(1051)
	PktUpdateSingleLED        = uint32(1052)
	PktSetCustomMode          = uint32(1100)
	PktUpdateMode             = uint32(1101)
	PktSaveMode               = uint32(1102)
	PktSignalUpdate           = uint32(1150)
)

const (
	ModeFlagHasSpeed          = uint32(1) << 0
	ModeFlagHasBrightness     = uint32(1) << 4
	ModeFlagHasPerLEDColor    = uint32(1) << 5
	ModeFlagHasModeSpecific   = uint32(1) << 6
	ModeFlagRequiresEntireDev = uint32(1) << 10
)

var ErrMalformed = errors.New("malformed openrgb packet")

type Header struct {
	DeviceID uint32
	PacketID uint32
	Size     uint32
}

func EncodeHeader(h Header) []byte {
	b := make([]byte, HeaderSize)
	copy(b[0:4], Magic)
	binary.LittleEndian.PutUint32(b[4:8], h.DeviceID)
	binary.LittleEndian.PutUint32(b[8:12], h.PacketID)
	binary.LittleEndian.PutUint32(b[12:16], h.Size)
	return b
}

func DecodeHeader(b []byte) (Header, error) {
	if len(b) < HeaderSize {
		return Header{}, fmt.Errorf("%w: header is %d bytes, want %d", ErrMalformed, len(b), HeaderSize)
	}
	if string(b[0:4]) != Magic {
		return Header{}, fmt.Errorf("%w: bad magic %q", ErrMalformed, string(b[0:4]))
	}
	h := Header{
		DeviceID: binary.LittleEndian.Uint32(b[4:8]),
		PacketID: binary.LittleEndian.Uint32(b[8:12]),
		Size:     binary.LittleEndian.Uint32(b[12:16]),
	}
	if h.Size > MaxPacketSize {
		return Header{}, fmt.Errorf("%w: payload of %d bytes exceeds the %d byte limit", ErrMalformed, h.Size, MaxPacketSize)
	}
	return h, nil
}

func Color(r, g, b uint8) uint32 {
	return uint32(r) | uint32(g)<<8 | uint32(b)<<16
}

func ColorParts(c uint32) (uint8, uint8, uint8) {
	return uint8(c & 0xFF), uint8((c >> 8) & 0xFF), uint8((c >> 16) & 0xFF)
}

func ParseHex(s string) (uint32, error) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) != 6 {
		return 0, fmt.Errorf("colour %q must be 6 hex digits, for example ff8800", s)
	}
	var vals [3]uint8
	for i := 0; i < 3; i++ {
		var v uint32
		if _, err := fmt.Sscanf(s[i*2:i*2+2], "%02x", &v); err != nil {
			return 0, fmt.Errorf("colour %q must be 6 hex digits, for example ff8800", s)
		}
		vals[i] = uint8(v)
	}
	return Color(vals[0], vals[1], vals[2]), nil
}

func HexString(c uint32) string {
	r, g, b := ColorParts(c)
	return fmt.Sprintf("%02x%02x%02x", r, g, b)
}

type buf struct {
	data []byte
}

func (b *buf) u16(v uint16) {
	var tmp [2]byte
	binary.LittleEndian.PutUint16(tmp[:], v)
	b.data = append(b.data, tmp[:]...)
}

func (b *buf) u32(v uint32) {
	var tmp [4]byte
	binary.LittleEndian.PutUint32(tmp[:], v)
	b.data = append(b.data, tmp[:]...)
}

func (b *buf) i32(v int32) {
	b.u32(uint32(v))
}

func (b *buf) str(s string) {
	b.u16(uint16(len(s) + 1))
	b.data = append(b.data, []byte(s)...)
	b.data = append(b.data, 0)
}

type cursor struct {
	data []byte
	pos  int
	err  error
}

func (c *cursor) fail(format string, args ...any) {
	if c.err == nil {
		c.err = fmt.Errorf("%w: %s", ErrMalformed, fmt.Sprintf(format, args...))
	}
}

func (c *cursor) take(n int) []byte {
	if c.err != nil {
		return nil
	}
	if n < 0 || c.pos+n > len(c.data) {
		c.fail("wanted %d bytes at offset %d but only %d remain", n, c.pos, len(c.data)-c.pos)
		return nil
	}
	out := c.data[c.pos : c.pos+n]
	c.pos += n
	return out
}

func (c *cursor) u16() uint16 {
	b := c.take(2)
	if b == nil {
		return 0
	}
	return binary.LittleEndian.Uint16(b)
}

func (c *cursor) u32() uint32 {
	b := c.take(4)
	if b == nil {
		return 0
	}
	return binary.LittleEndian.Uint32(b)
}

func (c *cursor) i32() int32 {
	return int32(c.u32())
}

func (c *cursor) str() string {
	n := int(c.u16())
	if c.err != nil {
		return ""
	}
	if n == 0 {
		return ""
	}
	b := c.take(n)
	if b == nil {
		return ""
	}
	return strings.TrimRight(string(b), "\x00")
}

type Mode struct {
	Name          string
	Value         int32
	Flags         uint32
	SpeedMin      uint32
	SpeedMax      uint32
	BrightnessMin uint32
	BrightnessMax uint32
	ColorsMin     uint32
	ColorsMax     uint32
	Speed         uint32
	Brightness    uint32
	Direction     uint32
	ColorMode     uint32
	Colors        []uint32
}

type Segment struct {
	Name     string
	Type     uint32
	StartIdx uint32
	LEDCount uint32
}

type Zone struct {
	Name      string
	Type      uint32
	LEDsMin   uint32
	LEDsMax   uint32
	LEDsCount uint32
	Matrix    []byte
	Segments  []Segment
	Flags     uint32
}

type LED struct {
	Name  string
	Value uint32
}

type Controller struct {
	Index       int
	Type        int32
	Name        string
	Vendor      string
	Description string
	Version     string
	Serial      string
	Location    string
	ActiveMode  int32
	Modes       []Mode
	Zones       []Zone
	LEDs        []LED
	Colors      []uint32
	Flags       uint32
}

func encodeMode(b *buf, m Mode, version uint32) {
	b.str(m.Name)
	b.i32(m.Value)
	b.u32(m.Flags)
	b.u32(m.SpeedMin)
	b.u32(m.SpeedMax)
	if version >= 3 {
		b.u32(m.BrightnessMin)
		b.u32(m.BrightnessMax)
	}
	b.u32(m.ColorsMin)
	b.u32(m.ColorsMax)
	b.u32(m.Speed)
	if version >= 3 {
		b.u32(m.Brightness)
	}
	b.u32(m.Direction)
	b.u32(m.ColorMode)
	b.u16(uint16(len(m.Colors)))
	for _, c := range m.Colors {
		b.u32(c)
	}
}

func decodeMode(c *cursor, version uint32) Mode {
	var m Mode
	m.Name = c.str()
	m.Value = c.i32()
	m.Flags = c.u32()
	m.SpeedMin = c.u32()
	m.SpeedMax = c.u32()
	if version >= 3 {
		m.BrightnessMin = c.u32()
		m.BrightnessMax = c.u32()
	}
	m.ColorsMin = c.u32()
	m.ColorsMax = c.u32()
	m.Speed = c.u32()
	if version >= 3 {
		m.Brightness = c.u32()
	}
	m.Direction = c.u32()
	m.ColorMode = c.u32()
	n := int(c.u16())
	if c.err != nil {
		return m
	}
	for i := 0; i < n; i++ {
		m.Colors = append(m.Colors, c.u32())
	}
	return m
}

func decodeZone(c *cursor, version uint32) Zone {
	var z Zone
	z.Name = c.str()
	z.Type = c.u32()
	z.LEDsMin = c.u32()
	z.LEDsMax = c.u32()
	z.LEDsCount = c.u32()
	matrixLen := int(c.u16())
	if matrixLen > 0 {
		if b := c.take(matrixLen); b != nil {
			z.Matrix = append([]byte(nil), b...)
		}
	}
	if version >= 4 {
		n := int(c.u16())
		if c.err != nil {
			return z
		}
		for i := 0; i < n; i++ {
			var s Segment
			s.Name = c.str()
			s.Type = c.u32()
			s.StartIdx = c.u32()
			s.LEDCount = c.u32()
			z.Segments = append(z.Segments, s)
		}
	}
	if version >= 5 {
		z.Flags = c.u32()
	}
	return z
}

func EncodeControllerData(ctrl Controller, version uint32) []byte {
	b := &buf{}
	b.i32(ctrl.Type)
	b.str(ctrl.Name)
	if version >= 1 {
		b.str(ctrl.Vendor)
	}
	b.str(ctrl.Description)
	b.str(ctrl.Version)
	b.str(ctrl.Serial)
	b.str(ctrl.Location)
	b.u16(uint16(len(ctrl.Modes)))
	b.i32(ctrl.ActiveMode)
	for _, m := range ctrl.Modes {
		encodeMode(b, m, version)
	}
	b.u16(uint16(len(ctrl.Zones)))
	for _, z := range ctrl.Zones {
		b.str(z.Name)
		b.u32(z.Type)
		b.u32(z.LEDsMin)
		b.u32(z.LEDsMax)
		b.u32(z.LEDsCount)
		b.u16(uint16(len(z.Matrix)))
		b.data = append(b.data, z.Matrix...)
		if version >= 4 {
			b.u16(uint16(len(z.Segments)))
			for _, s := range z.Segments {
				b.str(s.Name)
				b.u32(s.Type)
				b.u32(s.StartIdx)
				b.u32(s.LEDCount)
			}
		}
		if version >= 5 {
			b.u32(z.Flags)
		}
	}
	b.u16(uint16(len(ctrl.LEDs)))
	for _, l := range ctrl.LEDs {
		b.str(l.Name)
		b.u32(l.Value)
	}
	b.u16(uint16(len(ctrl.Colors)))
	for _, c := range ctrl.Colors {
		b.u32(c)
	}
	if version >= 5 {
		b.u16(0)
		b.u32(ctrl.Flags)
	}
	out := &buf{}
	out.u32(uint32(len(b.data)) + 4)
	out.data = append(out.data, b.data...)
	return out.data
}

func DecodeControllerData(data []byte, version uint32) (Controller, error) {
	c := &cursor{data: data}
	size := c.u32()
	if c.err != nil {
		return Controller{}, c.err
	}
	if int(size) != len(data) {
		return Controller{}, fmt.Errorf("%w: declared size %d but got %d bytes", ErrMalformed, size, len(data))
	}

	var ctrl Controller
	ctrl.Type = c.i32()
	ctrl.Name = c.str()
	if version >= 1 {
		ctrl.Vendor = c.str()
	}
	ctrl.Description = c.str()
	ctrl.Version = c.str()
	ctrl.Serial = c.str()
	ctrl.Location = c.str()

	numModes := int(c.u16())
	ctrl.ActiveMode = c.i32()
	if c.err != nil {
		return Controller{}, c.err
	}
	for i := 0; i < numModes; i++ {
		ctrl.Modes = append(ctrl.Modes, decodeMode(c, version))
		if c.err != nil {
			return Controller{}, c.err
		}
	}

	numZones := int(c.u16())
	if c.err != nil {
		return Controller{}, c.err
	}
	for i := 0; i < numZones; i++ {
		ctrl.Zones = append(ctrl.Zones, decodeZone(c, version))
		if c.err != nil {
			return Controller{}, c.err
		}
	}

	numLEDs := int(c.u16())
	if c.err != nil {
		return Controller{}, c.err
	}
	for i := 0; i < numLEDs; i++ {
		ctrl.LEDs = append(ctrl.LEDs, LED{Name: c.str(), Value: c.u32()})
		if c.err != nil {
			return Controller{}, c.err
		}
	}

	numColors := int(c.u16())
	if c.err != nil {
		return Controller{}, c.err
	}
	for i := 0; i < numColors; i++ {
		ctrl.Colors = append(ctrl.Colors, c.u32())
	}

	if version >= 5 {
		numAlt := int(c.u16())
		for i := 0; i < numAlt; i++ {
			c.str()
		}
		ctrl.Flags = c.u32()
	}

	if c.err != nil {
		return Controller{}, c.err
	}
	return ctrl, nil
}

func EncodeUpdateZoneLEDs(zoneIndex uint32, colors []uint32) []byte {
	body := &buf{}
	body.u32(zoneIndex)
	body.u16(uint16(len(colors)))
	for _, c := range colors {
		body.u32(c)
	}
	out := &buf{}
	out.u32(uint32(len(body.data)) + 4)
	out.data = append(out.data, body.data...)
	return out.data
}

func EncodeUpdateLEDs(colors []uint32) []byte {
	body := &buf{}
	body.u16(uint16(len(colors)))
	for _, c := range colors {
		body.u32(c)
	}
	out := &buf{}
	out.u32(uint32(len(body.data)) + 4)
	out.data = append(out.data, body.data...)
	return out.data
}

func EncodeUpdateMode(modeIndex int32, m Mode, version uint32) []byte {
	body := &buf{}
	body.i32(modeIndex)
	encodeMode(body, m, version)
	out := &buf{}
	out.u32(uint32(len(body.data)) + 4)
	out.data = append(out.data, body.data...)
	return out.data
}

func DecodeUpdateZoneLEDs(data []byte) (uint32, []uint32, error) {
	c := &cursor{data: data}
	size := c.u32()
	if c.err == nil && int(size) != len(data) {
		return 0, nil, fmt.Errorf("%w: declared size %d but got %d bytes", ErrMalformed, size, len(data))
	}
	zone := c.u32()
	n := int(c.u16())
	if c.err != nil {
		return 0, nil, c.err
	}
	colors := make([]uint32, 0, n)
	for i := 0; i < n; i++ {
		colors = append(colors, c.u32())
	}
	if c.err != nil {
		return 0, nil, c.err
	}
	return zone, colors, nil
}

func DecodeUpdateMode(data []byte, version uint32) (int32, Mode, error) {
	c := &cursor{data: data}
	size := c.u32()
	if c.err == nil && int(size) != len(data) {
		return 0, Mode{}, fmt.Errorf("%w: declared size %d but got %d bytes", ErrMalformed, size, len(data))
	}
	index := c.i32()
	if c.err != nil {
		return 0, Mode{}, c.err
	}
	m := decodeMode(c, version)
	if c.err != nil {
		return 0, Mode{}, c.err
	}
	return index, m, nil
}
