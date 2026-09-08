package kbd

import (
	"bytes"
	"testing"
)

func frame(fields map[int]byte) []byte {
	b := make([]byte, ReportLength)
	b[0] = 0xcc
	for pos, v := range fields {
		b[pos] = v
	}
	return b
}

func TestResetFrame(t *testing.T) {
	got := ResetFrame()
	want := frame(map[int]byte{1: 0x94})
	if !bytes.Equal(got, want) {
		t.Fatalf("ResetFrame() = % x, want % x", got, want)
	}
	if len(got) != ReportLength {
		t.Fatalf("frame length = %d, want %d", len(got), ReportLength)
	}
}

func TestStatusFrame(t *testing.T) {
	got := StatusFrame()
	want := frame(map[int]byte{1: 0x93})
	if !bytes.Equal(got, want) {
		t.Fatalf("StatusFrame() = % x, want % x", got, want)
	}
}

func TestColorSetFrameSingleKey(t *testing.T) {
	got := ColorSetFrame([]KeyColor{{Index: 5, R: 0x11, G: 0x22, B: 0x33}})
	want := frame(map[int]byte{1: 0x8c, 2: 0x02, 4: 6, 5: 0x11, 6: 0x22, 7: 0x33})
	if !bytes.Equal(got, want) {
		t.Fatalf("ColorSetFrame(single) = % x, want % x", got, want)
	}
}

func TestColorSetFrameKeyIndexIsOffByOneOnWire(t *testing.T) {
	got := ColorSetFrame([]KeyColor{{Index: 0, R: 1, G: 2, B: 3}})
	if got[4] != 1 {
		t.Fatalf("wire index for key 0 = %d, want 1", got[4])
	}
}

func TestColorSetFrameMultipleKeysStrideBy4(t *testing.T) {
	got := ColorSetFrame([]KeyColor{
		{Index: 0, R: 1},
		{Index: 1, R: 2},
		{Index: 2, R: 3},
	})
	if got[4] != 1 || got[5] != 1 {
		t.Fatalf("first block at offset 4 = %d,%d, want index=1,r=1", got[4], got[5])
	}
	if got[8] != 2 || got[9] != 2 {
		t.Fatalf("second block at offset 8 = %d,%d, want index=2,r=2", got[8], got[9])
	}
	if got[12] != 3 || got[13] != 3 {
		t.Fatalf("third block at offset 12 = %d,%d, want index=3,r=3", got[12], got[13])
	}
}

func TestColorSetFrameCapsAtFifteenKeys(t *testing.T) {
	if MaxKeysPerColorSetFrame != 15 {
		t.Fatalf("MaxKeysPerColorSetFrame = %d, want 15", MaxKeysPerColorSetFrame)
	}
	keys := make([]KeyColor, MaxKeysPerColorSetFrame+1)
	for i := range keys {
		keys[i] = KeyColor{Index: uint8(i), R: 0xff}
	}
	got := ColorSetFrame(keys)
	lastBlockStart := colorBlockStart + (MaxKeysPerColorSetFrame-1)*colorBlockSize
	if got[lastBlockStart] != uint8(MaxKeysPerColorSetFrame-1)+1 {
		t.Fatalf("the 15th block should still be written, block start %d", lastBlockStart)
	}
	overflowStart := colorBlockStart + MaxKeysPerColorSetFrame*colorBlockSize
	if overflowStart < ReportLength && got[overflowStart] != 0 {
		t.Fatalf("a 16th key must not be written to the wire, byte %d = %d", overflowStart, got[overflowStart])
	}
}

func TestLoopFrame(t *testing.T) {
	got := LoopFrame()
	want := frame(map[int]byte{1: 0x8c, 2: 0x13})
	if !bytes.Equal(got, want) {
		t.Fatalf("LoopFrame() = % x, want % x", got, want)
	}
}

func TestUpdateFrame(t *testing.T) {
	got := UpdateFrame()
	want := frame(map[int]byte{1: 0x8b, 2: 0x01, 3: 0xff})
	if !bytes.Equal(got, want) {
		t.Fatalf("UpdateFrame() = % x, want % x", got, want)
	}
}

func TestTurnOnFrame(t *testing.T) {
	got := TurnOnFrame(0x40)
	want := frame(map[int]byte{1: 0x83, 2: 0x38, 3: 0x9c, 4: 0x40})
	if !bytes.Equal(got, want) {
		t.Fatalf("TurnOnFrame() = % x, want % x", got, want)
	}
}

func TestStatusName(t *testing.T) {
	cases := map[uint8]string{
		0:                  "zero",
		StatusWaitUpdate:   "wait-update",
		StatusStartCommand: "start-command",
		StatusInCommand:    "in-command",
		0x01:               "unknown",
	}
	for status, want := range cases {
		if got := StatusName(status); got != want {
			t.Errorf("StatusName(%#x) = %q, want %q", status, got, want)
		}
	}
}
