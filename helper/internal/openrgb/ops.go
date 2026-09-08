package openrgb

import (
	"math"
	"os"
	"strconv"
	"strings"
	"time"
)

const DeviceIndexEnv = "ALIENWARECTL_RGB_DEVICE"

type StatusZone struct {
	Index    int    `json:"index"`
	Name     string `json:"name"`
	LEDCount int    `json:"ledCount"`
}

type StatusDevice struct {
	Index      int      `json:"index"`
	Name       string   `json:"name"`
	ZoneCount  int      `json:"zoneCount"`
	LEDCount   int      `json:"ledCount"`
	ActiveMode string   `json:"activeMode"`
	Modes      []string `json:"modes"`
}

type Status struct {
	OK        bool         `json:"ok"`
	Server    string       `json:"server"`
	Connected bool         `json:"connected"`
	Device    StatusDevice `json:"device"`
	Zones     []StatusZone `json:"zones"`
}

func DeviceIndex() int {
	raw := strings.TrimSpace(os.Getenv(DeviceIndexEnv))
	if raw == "" {
		return 0
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < 0 {
		return 0
	}
	return v
}

type Session struct {
	Client *Client
	Ctrl   Controller
}

func Open(addr string) (*Session, error) {
	c, err := Dial(addr)
	if err != nil {
		return nil, err
	}
	count, err := c.ControllerCount()
	if err != nil {
		c.Close()
		return nil, err
	}
	if count == 0 {
		c.Close()
		return nil, newError(CodeNoOpenRGB, "the OpenRGB SDK server reports no controllers")
	}
	index := DeviceIndex()
	if index >= count {
		c.Close()
		return nil, newError(CodeBadRequest, "device %d does not exist, the server reports %d", index, count)
	}
	ctrl, err := c.Controller(index)
	if err != nil {
		c.Close()
		return nil, err
	}
	return &Session{Client: c, Ctrl: ctrl}, nil
}

func (s *Session) Close() error {
	return s.Client.Close()
}

func (s *Session) Status() Status {
	modes := make([]string, 0, len(s.Ctrl.Modes))
	for _, m := range s.Ctrl.Modes {
		modes = append(modes, m.Name)
	}
	active := ""
	if s.Ctrl.ActiveMode >= 0 && int(s.Ctrl.ActiveMode) < len(s.Ctrl.Modes) {
		active = s.Ctrl.Modes[s.Ctrl.ActiveMode].Name
	}
	zones := make([]StatusZone, 0, len(s.Ctrl.Zones))
	for i, z := range s.Ctrl.Zones {
		zones = append(zones, StatusZone{Index: i, Name: z.Name, LEDCount: int(z.LEDsCount)})
	}
	return Status{
		OK:        true,
		Server:    s.Client.Addr(),
		Connected: true,
		Device: StatusDevice{
			Index:      s.Ctrl.Index,
			Name:       s.Ctrl.Name,
			ZoneCount:  len(s.Ctrl.Zones),
			LEDCount:   len(s.Ctrl.LEDs),
			ActiveMode: active,
			Modes:      modes,
		},
		Zones: zones,
	}
}

func ResolveZone(ctrl Controller, arg string) (int, error) {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return 0, newError(CodeBadRequest, "a zone index or name is required")
	}
	if v, err := strconv.Atoi(arg); err == nil {
		if v < 0 || v >= len(ctrl.Zones) {
			return 0, newError(CodeBadRequest, "zone %d does not exist, the device has %d zones", v, len(ctrl.Zones))
		}
		return v, nil
	}
	want := strings.ToLower(arg)
	matches := []int{}
	for i, z := range ctrl.Zones {
		if strings.ToLower(z.Name) == want {
			return i, nil
		}
		if strings.Contains(strings.ToLower(z.Name), want) {
			matches = append(matches, i)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return 0, newError(CodeBadRequest, "zone %q is ambiguous, it matches %d zones", arg, len(matches))
	}
	return 0, newError(CodeBadRequest, "no zone called %q on this device", arg)
}

func ModeIndex(ctrl Controller, name string) (int, error) {
	want := strings.ToLower(strings.TrimSpace(name))
	if want == "" {
		return 0, newError(CodeBadRequest, "a mode name is required")
	}
	for i, m := range ctrl.Modes {
		if strings.ToLower(m.Name) == want {
			return i, nil
		}
	}
	names := make([]string, 0, len(ctrl.Modes))
	for _, m := range ctrl.Modes {
		names = append(names, m.Name)
	}
	return 0, newError(CodeNotSupported, "mode %q is not offered by this device, it has %s", name, strings.Join(names, ", "))
}

func (s *Session) ensurePerLEDMode() error {
	active := int(s.Ctrl.ActiveMode)
	if active >= 0 && active < len(s.Ctrl.Modes) && s.Ctrl.Modes[active].Flags&ModeFlagHasPerLEDColor != 0 {
		return nil
	}
	for i, m := range s.Ctrl.Modes {
		if m.Flags&ModeFlagHasPerLEDColor == 0 {
			continue
		}
		if err := s.Client.UpdateMode(s.Ctrl.Index, int32(i), m); err != nil {
			return err
		}
		s.Ctrl.ActiveMode = int32(i)
		return nil
	}
	return nil
}

func (s *Session) SetZone(index int, color uint32) error {
	if index < 0 || index >= len(s.Ctrl.Zones) {
		return newError(CodeBadRequest, "zone %d does not exist", index)
	}
	if err := s.ensurePerLEDMode(); err != nil {
		return err
	}
	count := int(s.Ctrl.Zones[index].LEDsCount)
	if count <= 0 {
		return newError(CodeNotSupported, "zone %d has no LEDs", index)
	}
	colors := make([]uint32, count)
	for i := range colors {
		colors[i] = color
	}
	return s.Client.UpdateZoneLEDs(s.Ctrl.Index, uint32(index), colors)
}

func (s *Session) SetAll(color uint32) error {
	if err := s.ensurePerLEDMode(); err != nil {
		return err
	}
	count := len(s.Ctrl.LEDs)
	if count == 0 {
		return newError(CodeNotSupported, "the device reports no LEDs")
	}
	colors := make([]uint32, count)
	for i := range colors {
		colors[i] = color
	}
	return s.Client.UpdateLEDs(s.Ctrl.Index, colors)
}

func (s *Session) SetMode(name string) (string, error) {
	index, err := ModeIndex(s.Ctrl, name)
	if err != nil {
		return "", err
	}
	if err := s.Client.UpdateMode(s.Ctrl.Index, int32(index), s.Ctrl.Modes[index]); err != nil {
		return "", err
	}
	s.Ctrl.ActiveMode = int32(index)
	return s.Ctrl.Modes[index].Name, nil
}

func ScaleBrightness(m Mode, percent int) uint32 {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	lo, hi := m.BrightnessMin, m.BrightnessMax
	if hi <= lo {
		return hi
	}
	span := float64(hi - lo)
	return lo + uint32(math.Round(span*float64(percent)/100.0))
}

func (s *Session) SetBrightness(percent int) (uint32, error) {
	if percent < 0 || percent > 100 {
		return 0, newError(CodeBadRequest, "brightness %d is out of range 0-100", percent)
	}
	active := int(s.Ctrl.ActiveMode)
	if active < 0 || active >= len(s.Ctrl.Modes) {
		return 0, newError(CodeNotSupported, "the device does not report an active mode")
	}
	m := s.Ctrl.Modes[active]
	if m.Flags&ModeFlagHasBrightness == 0 {
		return 0, newError(CodeNotSupported, "mode %q does not support brightness", m.Name)
	}
	m.Brightness = ScaleBrightness(m, percent)
	if err := s.Client.UpdateMode(s.Ctrl.Index, int32(active), m); err != nil {
		return 0, err
	}
	s.Ctrl.Modes[active] = m
	return m.Brightness, nil
}

func (s *Session) Off() error {
	return s.SetAll(0)
}

func (s *Session) Identify(index int, blinks int, period time.Duration) error {
	if index < 0 || index >= len(s.Ctrl.Zones) {
		return newError(CodeBadRequest, "zone %d does not exist", index)
	}
	saved := s.savedColors()
	red := Color(255, 0, 0)
	for i := 0; i < blinks; i++ {
		if err := s.SetZone(index, red); err != nil {
			return err
		}
		time.Sleep(period)
		if err := s.SetZone(index, 0); err != nil {
			return err
		}
		time.Sleep(period)
	}
	if saved == nil {
		return nil
	}
	return s.Client.UpdateLEDs(s.Ctrl.Index, saved)
}

func (s *Session) savedColors() []uint32 {
	if len(s.Ctrl.Colors) != len(s.Ctrl.LEDs) || len(s.Ctrl.Colors) == 0 {
		return nil
	}
	return append([]uint32(nil), s.Ctrl.Colors...)
}

func (s *Session) ZoneSummary(index int) (StatusZone, error) {
	if index < 0 || index >= len(s.Ctrl.Zones) {
		return StatusZone{}, newError(CodeBadRequest, "zone %d does not exist", index)
	}
	z := s.Ctrl.Zones[index]
	return StatusZone{Index: index, Name: z.Name, LEDCount: int(z.LEDsCount)}, nil
}
