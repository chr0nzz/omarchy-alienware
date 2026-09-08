package daemon

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/godbus/dbus/v5"

	"github.com/chr0nzz/omarchy-alienware/helper/internal/kbd"
)

type KeyboardStatus struct {
	OK       bool `json:"ok"`
	Present  bool `json:"present"`
	KeyCount int  `json:"keyCount,omitempty"`
}

func (s *Service) KeyboardStatus() (string, *dbus.Error) {
	out := KeyboardStatus{OK: true, Present: s.keyboardPresent()}
	if out.Present {
		out.KeyCount = kbd.KeyCount()
	}
	b, err := json.Marshal(out)
	if err != nil {
		return "", mapError(err)
	}
	return string(b), nil
}

func (s *Service) applyKeyboard(fn func(*kbd.Device) error) error {
	s.kbdMu.Lock()
	defer s.kbdMu.Unlock()
	dev, err := s.openKeyboard()
	if err != nil {
		return err
	}
	defer dev.Close()
	return fn(dev)
}

func (s *Service) SetKeyboardKeys(payload string, sender dbus.Sender) *dbus.Error {
	if err := s.auth.Authorize(sender, ActionSetKeyboard); err != nil {
		return mapError(err)
	}
	entries, err := parseKeyColorMap(payload)
	if err != nil {
		return dbusError(ErrNameBadRequest, err.Error())
	}
	return mapError(s.applyKeyboard(func(dev *kbd.Device) error {
		return dev.SetKeyColors(entries)
	}))
}

func (s *Service) SetKeyboardAll(color string, sender dbus.Sender) *dbus.Error {
	if err := s.auth.Authorize(sender, ActionSetKeyboard); err != nil {
		return mapError(err)
	}
	r, g, b, err := parseHexColor6(color)
	if err != nil {
		return dbusError(ErrNameBadRequest, err.Error())
	}
	return mapError(s.applyKeyboard(func(dev *kbd.Device) error {
		return dev.SetAllKeys(r, g, b, kbd.DefaultKeyFirst, kbd.DefaultKeyLast)
	}))
}

func (s *Service) KeyboardOff(sender dbus.Sender) *dbus.Error {
	if err := s.auth.Authorize(sender, ActionSetKeyboard); err != nil {
		return mapError(err)
	}
	return mapError(s.applyKeyboard(func(dev *kbd.Device) error {
		return dev.SetAllKeys(0, 0, 0, kbd.DefaultKeyFirst, kbd.DefaultKeyLast)
	}))
}

func parseHexColor6(s string) (uint8, uint8, uint8, error) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) != 6 {
		return 0, 0, 0, fmt.Errorf("colour %q must be 6 hex digits, for example ff8800", s)
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("colour %q must be 6 hex digits, for example ff8800", s)
	}
	return uint8(v >> 16), uint8(v >> 8), uint8(v), nil
}

func parseKeyColorMap(s string) ([]kbd.KeyColor, error) {
	parts := strings.Split(s, ",")
	seen := map[uint8]bool{}
	entries := make([]kbd.KeyColor, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 || strings.TrimSpace(kv[0]) == "" {
			return nil, fmt.Errorf("keyboard map entries must be idx=RRGGBB, got %q", part)
		}
		idx, err := strconv.Atoi(strings.TrimSpace(kv[0]))
		if err != nil || idx < kbd.DefaultKeyFirst || idx > kbd.DefaultKeyLast {
			return nil, fmt.Errorf("key index must be %d-%d, got %q", kbd.DefaultKeyFirst, kbd.DefaultKeyLast, kv[0])
		}
		r, g, b, cerr := parseHexColor6(kv[1])
		if cerr != nil {
			return nil, cerr
		}
		if seen[uint8(idx)] {
			return nil, fmt.Errorf("key %d is repeated", idx)
		}
		seen[uint8(idx)] = true
		entries = append(entries, kbd.KeyColor{Index: uint8(idx), R: r, G: g, B: b})
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("keyboard map needs at least one idx=RRGGBB pair")
	}
	return entries, nil
}
