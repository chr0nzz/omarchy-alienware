package daemon

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/chr0nzz/omarchy-alienware/helper/internal/fan"
)

func hotCurve() fan.Curve {
	return fan.Curve{
		Interval:   1,
		Hysteresis: 0,
		CPU:        []fan.Point{{Temp: 0, Boost: 200}, {Temp: 110, Boost: 200}},
		GPU:        []fan.Point{{Temp: 0, Boost: 200}, {Temp: 110, Boost: 200}},
	}
}

func boostValue(t *testing.T, root string, index int) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, "sys/class/hwmon/hwmon4", "fan"+string(rune('0'+index))+"_boost"))
	if err != nil {
		t.Fatalf("reading boost: %v", err)
	}
	return strings.TrimSpace(string(b))
}

func TestConcurrentApplyCurveLeavesNoOrphanLoop(t *testing.T) {
	for attempt := 0; attempt < 60; attempt++ {
		reader := fakeReader(t)
		svc := NewService(reader, nil, quietLogger())

		var wg sync.WaitGroup
		for i := 0; i < 4; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := svc.StartCurve(hotCurve()); err != nil {
					t.Errorf("StartCurve: %v", err)
				}
			}()
		}
		wg.Wait()
		svc.Stop()

		if err := reader.ResetBoost(); err != nil {
			t.Fatalf("reset: %v", err)
		}
		time.Sleep(40 * time.Millisecond)

		for _, index := range []int{1, 2} {
			if got := boostValue(t, reader.Root, index); got != "0" {
				t.Fatalf("attempt %d: an orphaned curve loop rewrote fan%d_boost to %q after Stop", attempt, index, got)
			}
		}
	}
}

func TestStopDoesNotUndoAProfileTheUserChose(t *testing.T) {
	reader := fakeReader(t)
	svc := NewService(reader, nil, quietLogger())

	if err := svc.StartCurve(hotCurve()); err != nil {
		t.Fatalf("StartCurve: %v", err)
	}
	if err := reader.SetProfile("performance"); err != nil {
		t.Fatalf("SetProfile: %v", err)
	}
	svc.Stop()

	got, err := reader.ReadProfile()
	if err != nil {
		t.Fatalf("ReadProfile: %v", err)
	}
	if got != "performance" {
		t.Fatalf("a profile chosen while the curve ran must survive the stop, got %q", got)
	}
}

func TestStopRestoresTheProfileTheCurveReplaced(t *testing.T) {
	reader := fakeReader(t)
	svc := NewService(reader, nil, quietLogger())

	if err := svc.StartCurve(hotCurve()); err != nil {
		t.Fatalf("StartCurve: %v", err)
	}
	current, err := reader.ReadProfile()
	if err != nil {
		t.Fatalf("ReadProfile: %v", err)
	}
	if current != customProfile {
		t.Fatalf("a running curve must select %q, got %q", customProfile, current)
	}
	svc.Stop()

	got, err := reader.ReadProfile()
	if err != nil {
		t.Fatalf("ReadProfile: %v", err)
	}
	if got != "balanced" {
		t.Fatalf("stopping a curve must restore the previous profile, got %q", got)
	}
}
