package openrgb

import (
	"errors"
	"testing"
	"time"
)

func brightnessController(min, max, active uint32) Controller {
	return Controller{
		Name:       "Dell G Series LED Controller",
		ActiveMode: 0,
		Modes: []Mode{
			{Name: "Static", Flags: ModeFlagHasPerLEDColor | ModeFlagHasBrightness, BrightnessMin: min, BrightnessMax: max, Brightness: active},
		},
		Zones:  []Zone{{Name: "Zone 0", LEDsMin: 1, LEDsMax: 1, LEDsCount: 1}},
		LEDs:   []LED{{Name: "LED 0"}},
		Colors: []uint32{0},
	}
}

func TestSetBrightnessScalesRealRange(t *testing.T) {
	cases := []struct {
		percent int
		want    uint32
	}{
		{0, 10},
		{50, 15},
		{100, 20},
	}
	for _, c := range cases {
		session, server := openFakeSession(t, brightnessController(10, 20, 10))
		got, err := session.SetBrightness(c.percent)
		if err != nil {
			t.Fatalf("percent %d: SetBrightness: %v", c.percent, err)
		}
		if got != c.want {
			t.Fatalf("percent %d: got %d, want %d", c.percent, got, c.want)
		}
		waitForCount(t, server.updateModeCount, 1, time.Second)
	}
}

func TestSetBrightnessRejectsDegenerateZeroRange(t *testing.T) {
	session, server := openFakeSession(t, brightnessController(0, 0, 0))
	_, err := session.SetBrightness(100)
	if err == nil {
		t.Fatal("want an error for a degenerate 0..0 brightness range")
	}
	var oerr *Error
	if !errors.As(err, &oerr) || oerr.Code() != CodeNotSupported {
		t.Fatalf("got %v, want a %q error", err, CodeNotSupported)
	}
	time.Sleep(75 * time.Millisecond)
	if got := server.updateModeCount(); got != 0 {
		t.Fatalf("a rejected degenerate brightness must never reach UPDATEMODE, got %d calls", got)
	}
}

func TestSetBrightnessAllowsNonzeroFlatRange(t *testing.T) {
	session, server := openFakeSession(t, brightnessController(50, 50, 0))
	got, err := session.SetBrightness(70)
	if err != nil {
		t.Fatalf("SetBrightness: %v", err)
	}
	if got != 50 {
		t.Fatalf("got %d, want 50", got)
	}
	waitForCount(t, server.updateModeCount, 1, time.Second)
}
