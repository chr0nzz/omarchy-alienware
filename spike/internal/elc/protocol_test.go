package elc

import (
	"bytes"
	"testing"
)

func frame(fields map[int]byte) []byte {
	b := make([]byte, ReportLength)
	for pos, v := range fields {
		b[pos] = v
	}
	return b
}

func TestControlFrame(t *testing.T) {
	got := ControlFrame(ControlFinishPlay, ControlIDCommon)
	want := frame(map[int]byte{1: 0x03, 2: 0x21, 4: 3, 5: 0xff, 6: 0xff})
	if !bytes.Equal(got, want) {
		t.Fatalf("ControlFrame(finish-play, common) = % x, want % x", got, want)
	}
	if len(got) != ReportLength {
		t.Fatalf("frame length = %d, want %d", len(got), ReportLength)
	}
}

func TestControlFrameResetSequence(t *testing.T) {
	remove := ControlFrame(ControlRemove, ControlIDCommon)
	wantRemove := frame(map[int]byte{1: 0x03, 2: 0x21, 4: 4, 5: 0xff, 6: 0xff})
	if !bytes.Equal(remove, wantRemove) {
		t.Fatalf("remove frame = % x, want % x", remove, wantRemove)
	}

	start := ControlFrame(ControlStartNew, ControlIDCommon)
	wantStart := frame(map[int]byte{1: 0x03, 2: 0x21, 4: 1, 5: 0xff, 6: 0xff})
	if !bytes.Equal(start, wantStart) {
		t.Fatalf("start frame = % x, want % x", start, wantStart)
	}
}

func TestControlFrameNamedControlID(t *testing.T) {
	got := ControlFrame(ControlRemove, ControlIDLight)
	want := frame(map[int]byte{1: 0x03, 2: 0x21, 4: 4, 5: 0x00, 6: 0x61})
	if !bytes.Equal(got, want) {
		t.Fatalf("ControlFrame(remove, light) = % x, want % x", got, want)
	}
}

func TestSelectZonesFrameSingleZone(t *testing.T) {
	got := SelectZonesFrame(true, []uint8{5})
	want := frame(map[int]byte{1: 0x03, 2: 0x23, 3: 1, 5: 1, 6: 5})
	if !bytes.Equal(got, want) {
		t.Fatalf("SelectZonesFrame(loop, [5]) = % x, want % x", got, want)
	}
}

func TestSelectZonesFrameNoLoop(t *testing.T) {
	got := SelectZonesFrame(false, []uint8{2, 4})
	want := frame(map[int]byte{1: 0x03, 2: 0x23, 5: 2, 6: 2, 7: 4})
	if !bytes.Equal(got, want) {
		t.Fatalf("SelectZonesFrame(once, [2,4]) = % x, want % x", got, want)
	}
}

func TestAddActionFrameSingleColorPhase(t *testing.T) {
	got := AddActionFrame([]Phase{{Type: ActionColor, Time: 7, Tempo: 0, R: 0x11, G: 0x22, B: 0x33}})
	want := frame(map[int]byte{
		1: 0x03, 2: 0x24,
		3: uint8(ActionColor), 4: 7, 5: actionOpCodes[ActionColor], 7: 0xfa,
		8: 0x11, 9: 0x22, 10: 0x33,
	})
	if !bytes.Equal(got, want) {
		t.Fatalf("AddActionFrame(color) = % x, want % x", got, want)
	}
}

func TestAddActionFrameTempoKeptForNonColorPhase(t *testing.T) {
	got := AddActionFrame([]Phase{{Type: ActionPulse, Time: 3, Tempo: 0x40, R: 1, G: 2, B: 3}})
	want := frame(map[int]byte{
		1: 0x03, 2: 0x24,
		3: uint8(ActionPulse), 4: 3, 5: actionOpCodes[ActionPulse], 7: 0x40,
		8: 1, 9: 2, 10: 3,
	})
	if !bytes.Equal(got, want) {
		t.Fatalf("AddActionFrame(pulse) = % x, want % x", got, want)
	}
}

func TestAddActionFrameHighActionTypesClampToMorph(t *testing.T) {
	got := AddActionFrame([]Phase{{Type: ActionSpectrum, Time: 1, Tempo: 5, R: 9, G: 9, B: 9}})
	if got[3] != uint8(ActionMorph) {
		t.Fatalf("frame type byte = %d, want %d (ActionMorph)", got[3], ActionMorph)
	}
	if got[5] != actionOpCodes[ActionSpectrum] {
		t.Fatalf("mode byte should still use the unclamped action's opcode, got %#x, want %#x", got[5], actionOpCodes[ActionSpectrum])
	}
}

func TestAddActionFrameMultiplePhasesStrideBy8(t *testing.T) {
	got := AddActionFrame([]Phase{
		{Type: ActionColor, R: 1},
		{Type: ActionColor, R: 2},
		{Type: ActionColor, R: 3},
	})
	if got[8] != 1 {
		t.Fatalf("first phase r at offset 8 = %d, want 1", got[8])
	}
	if got[16] != 2 {
		t.Fatalf("second phase r at offset 16 = %d, want 2", got[16])
	}
	if got[24] != 3 {
		t.Fatalf("third phase r at offset 24 = %d, want 3", got[24])
	}
}

func TestAddActionFrameCapsAtThreePhases(t *testing.T) {
	got := AddActionFrame([]Phase{
		{Type: ActionColor, R: 1},
		{Type: ActionColor, R: 2},
		{Type: ActionColor, R: 3},
		{Type: ActionColor, R: 4},
	})
	for i := 27; i < ReportLength; i++ {
		if got[i] != 0 {
			t.Fatalf("byte %d = %d, want 0: a fourth phase must not be written to the wire", i, got[i])
		}
	}
}

func TestSetOneColorFrame(t *testing.T) {
	got := SetOneColorFrame(0xaa, 0xbb, 0xcc, []uint8{5})
	want := frame(map[int]byte{1: 0x03, 2: 0x27, 3: 0xaa, 4: 0xbb, 5: 0xcc, 7: 1, 8: 5})
	if !bytes.Equal(got, want) {
		t.Fatalf("SetOneColorFrame = % x, want % x", got, want)
	}
}

func TestSetOneColorFrameMultipleZones(t *testing.T) {
	got := SetOneColorFrame(1, 2, 3, []uint8{10, 11, 12})
	want := frame(map[int]byte{1: 0x03, 2: 0x27, 3: 1, 4: 2, 5: 3, 7: 3, 8: 10, 9: 11, 10: 12})
	if !bytes.Equal(got, want) {
		t.Fatalf("SetOneColorFrame(multi) = % x, want % x", got, want)
	}
}

func TestTurnOnFrame(t *testing.T) {
	got := TurnOnFrame(36, []uint8{1, 2, 3})
	want := frame(map[int]byte{1: 0x03, 2: 0x26, 3: 36, 5: 3, 6: 1, 7: 2, 8: 3})
	if !bytes.Equal(got, want) {
		t.Fatalf("TurnOnFrame = % x, want % x", got, want)
	}
}

func TestSetPowerControlFrame(t *testing.T) {
	got := SetPowerControlFrame(ControlRemove, 0x5b)
	want := frame(map[int]byte{1: 0x03, 2: 0x22, 4: 4, 6: 0x5b})
	if !bytes.Equal(got, want) {
		t.Fatalf("SetPowerControlFrame = % x, want % x", got, want)
	}
}

func TestStatusName(t *testing.T) {
	cases := map[uint8]string{
		0:                  "zero",
		StatusV4Ready:      "ready",
		StatusV4Busy:       "busy",
		StatusV4WaitColor:  "wait-color",
		StatusV4WaitUpdate: "wait-update",
		StatusV4WasOn:      "was-on",
		200:                "unknown",
	}
	for status, want := range cases {
		if got := StatusName(status); got != want {
			t.Errorf("StatusName(%d) = %q, want %q", status, got, want)
		}
	}
}
