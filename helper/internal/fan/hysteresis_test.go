package fan

import "testing"

func TestHysteresisRisingAppliesImmediately(t *testing.T) {
	p := pts(40, 0, 90, 250)
	s := NewState(5)
	if got := s.Update(p, 40); got != 0 {
		t.Fatalf("first sample: got %d, want 0", got)
	}
	if got := s.Update(p, 50); got != 50 {
		t.Fatalf("rise to 50: got %d, want 50", got)
	}
	if got := s.Update(p, 65); got != 125 {
		t.Fatalf("rise to 65: got %d, want 125", got)
	}
}

func TestHysteresisDampsFalling(t *testing.T) {
	p := pts(40, 0, 90, 250)
	s := NewState(5)
	s.Update(p, 65)
	if got := s.Update(p, 62); got != 125 {
		t.Fatalf("small fall should hold: got %d, want 125", got)
	}
	if got := s.Update(p, 60); got != 125 {
		t.Fatalf("fall of exactly the hysteresis should hold: got %d, want 125", got)
	}
	if got := s.Update(p, 59); got != 95 {
		t.Fatalf("fall past the hysteresis should apply: got %d, want 95", got)
	}
}

func TestHysteresisZeroTracksImmediately(t *testing.T) {
	p := pts(40, 0, 90, 250)
	s := NewState(0)
	s.Update(p, 65)
	if got := s.Update(p, 64); got != 120 {
		t.Fatalf("zero hysteresis: got %d, want 120", got)
	}
}

func TestHysteresisOscillationAroundThreshold(t *testing.T) {
	p := pts(50, 0, 51, 200, 90, 255)
	s := NewState(3)
	if got := s.Update(p, 50); got != 0 {
		t.Fatalf("start: got %d, want 0", got)
	}
	if got := s.Update(p, 51); got != 200 {
		t.Fatalf("cross up: got %d, want 200", got)
	}
	for i, temp := range []float64{50, 51, 50, 51, 49, 51, 48.5} {
		if got := s.Update(p, temp); got != 200 {
			t.Fatalf("oscillation step %d at %.1f: got %d, want 200", i, temp, got)
		}
	}
	if got := s.Update(p, 47); got != 0 {
		t.Fatalf("clear fall past the hysteresis: got %d, want 0", got)
	}
}

func TestHysteresisReleasesFromLatestHeldTemp(t *testing.T) {
	p := pts(40, 0, 140, 100)
	s := NewState(4)
	s.Update(p, 80)
	if got := s.Update(p, 77); got != 40 {
		t.Fatalf("held boost: got %d, want 40", got)
	}
	if got := s.Update(p, 75); got != 35 {
		t.Fatalf("released: got %d, want 35", got)
	}
	if got := s.Update(p, 72); got != 35 {
		t.Fatalf("hold again from the new held temp: got %d, want 35", got)
	}
	if got := s.Update(p, 70); got != 30 {
		t.Fatalf("release again: got %d, want 30", got)
	}
}

func TestHysteresisRisingUpdatesHeldTemp(t *testing.T) {
	p := pts(40, 0, 140, 100)
	s := NewState(5)
	s.Update(p, 60)
	s.Update(p, 90)
	if s.HeldTemp() != 90 {
		t.Fatalf("held temp: got %.1f, want 90", s.HeldTemp())
	}
	if got := s.Update(p, 86); got != 50 {
		t.Fatalf("hold after a rise: got %d, want 50", got)
	}
	if got := s.Update(p, 84); got != 44 {
		t.Fatalf("release after a rise: got %d, want 44", got)
	}
}

func TestHysteresisResetRepriming(t *testing.T) {
	p := pts(40, 0, 90, 250)
	s := NewState(10)
	s.Update(p, 80)
	if s.Boost() != 200 {
		t.Fatalf("boost: got %d, want 200", s.Boost())
	}
	s.Reset()
	if s.Boost() != 0 {
		t.Fatalf("after reset: got %d, want 0", s.Boost())
	}
	if got := s.Update(p, 45); got != 25 {
		t.Fatalf("reprimed low: got %d, want 25", got)
	}
}

func TestNewStateClampsHysteresis(t *testing.T) {
	p := pts(0, 0, 110, 110)
	s := NewState(999)
	s.Update(p, 100)
	if got := s.Update(p, 90); got != 100 {
		t.Fatalf("clamped hysteresis should hold: got %d, want 100", got)
	}
	if got := s.Update(p, 84); got != 84 {
		t.Fatalf("beyond the clamped hysteresis: got %d, want 84", got)
	}
}
