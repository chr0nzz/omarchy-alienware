package fan

import (
	"errors"
	"reflect"
	"testing"
)

func pts(v ...int) []Point {
	out := make([]Point, 0, len(v)/2)
	for i := 0; i+1 < len(v); i += 2 {
		out = append(out, Point{Temp: v[i], Boost: v[i+1]})
	}
	return out
}

func TestInterpolateEndpoints(t *testing.T) {
	p := pts(40, 0, 90, 255)
	if got := Interpolate(p, 10); got != 0 {
		t.Fatalf("below first point: got %d, want 0", got)
	}
	if got := Interpolate(p, 40); got != 0 {
		t.Fatalf("at first point: got %d, want 0", got)
	}
	if got := Interpolate(p, 90); got != 255 {
		t.Fatalf("at last point: got %d, want 255", got)
	}
	if got := Interpolate(p, 120); got != 255 {
		t.Fatalf("above last point: got %d, want 255", got)
	}
}

func TestInterpolateMidpoint(t *testing.T) {
	p := pts(40, 0, 90, 250)
	if got := Interpolate(p, 65); got != 125 {
		t.Fatalf("midpoint: got %d, want 125", got)
	}
	if got := Interpolate(p, 50); got != 50 {
		t.Fatalf("quarter point: got %d, want 50", got)
	}
}

func TestInterpolateMultiSegment(t *testing.T) {
	p := pts(30, 0, 50, 60, 70, 200, 90, 255)
	cases := map[float64]int{
		30: 0,
		40: 30,
		50: 60,
		60: 130,
		70: 200,
		80: 228,
		90: 255,
	}
	for temp, want := range cases {
		if got := Interpolate(p, temp); got != want {
			t.Errorf("temp %.0f: got %d, want %d", temp, got, want)
		}
	}
}

func TestInterpolateDescendingSegment(t *testing.T) {
	p := pts(40, 200, 80, 100)
	if got := Interpolate(p, 60); got != 150 {
		t.Fatalf("descending segment: got %d, want 150", got)
	}
}

func TestInterpolateEmpty(t *testing.T) {
	if got := Interpolate(nil, 50); got != 0 {
		t.Fatalf("empty curve: got %d, want 0", got)
	}
}

func TestNormalizeSortsAndClamps(t *testing.T) {
	in := Curve{
		Interval:   99,
		Hysteresis: -4,
		CPU:        pts(90, 900, 40, -20),
		GPU:        pts(200, 10, 30, 5),
	}
	out, err := Normalize(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Interval != MaxInterval {
		t.Errorf("interval: got %d, want %d", out.Interval, MaxInterval)
	}
	if out.Hysteresis != MinHysteresis {
		t.Errorf("hysteresis: got %d, want %d", out.Hysteresis, MinHysteresis)
	}
	wantCPU := pts(40, 0, 90, 255)
	if !reflect.DeepEqual(out.CPU, wantCPU) {
		t.Errorf("cpu: got %v, want %v", out.CPU, wantCPU)
	}
	wantGPU := pts(30, 5, 110, 10)
	if !reflect.DeepEqual(out.GPU, wantGPU) {
		t.Errorf("gpu: got %v, want %v", out.GPU, wantGPU)
	}
}

func TestNormalizeDedupesDuplicateTemps(t *testing.T) {
	out, err := Normalize(Curve{Interval: 2, Hysteresis: 3,
		CPU: pts(40, 10, 40, 90, 80, 200),
		GPU: pts(40, 0, 80, 200),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := pts(40, 90, 80, 200)
	if !reflect.DeepEqual(out.CPU, want) {
		t.Fatalf("cpu: got %v, want %v", out.CPU, want)
	}
}

func TestNormalizeRejectsTooFewPoints(t *testing.T) {
	_, err := Normalize(Curve{Interval: 2, CPU: pts(40, 0), GPU: pts(40, 0, 80, 200)})
	if !errors.Is(err, ErrInvalidCurve) {
		t.Fatalf("want ErrInvalidCurve, got %v", err)
	}
}

func TestNormalizeRejectsCollapsedDuplicates(t *testing.T) {
	_, err := Normalize(Curve{Interval: 2, CPU: pts(50, 0, 50, 200), GPU: pts(40, 0, 80, 200)})
	if !errors.Is(err, ErrInvalidCurve) {
		t.Fatalf("want ErrInvalidCurve, got %v", err)
	}
}

func TestNormalizeRejectsTooManyPoints(t *testing.T) {
	big := pts(10, 0, 20, 10, 30, 20, 40, 30, 50, 40, 60, 50, 70, 60, 80, 70, 90, 80)
	_, err := Normalize(Curve{Interval: 2, CPU: big, GPU: pts(40, 0, 80, 200)})
	if !errors.Is(err, ErrInvalidCurve) {
		t.Fatalf("want ErrInvalidCurve, got %v", err)
	}
}

func TestNormalizeRejectsMissingSide(t *testing.T) {
	_, err := Normalize(Curve{Interval: 2, CPU: pts(40, 0, 80, 200)})
	if !errors.Is(err, ErrInvalidCurve) {
		t.Fatalf("want ErrInvalidCurve, got %v", err)
	}
}

func TestParseValid(t *testing.T) {
	raw := []byte(`{"interval":2,"hysteresis":3,
		"cpu":[{"temp":40,"boost":0},{"temp":90,"boost":255}],
		"gpu":[{"temp":40,"boost":0},{"temp":90,"boost":255}]}`)
	c, err := Parse(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Interval != 2 || c.Hysteresis != 3 {
		t.Fatalf("got interval %d hysteresis %d", c.Interval, c.Hysteresis)
	}
	if len(c.CPU) != 2 || len(c.GPU) != 2 {
		t.Fatalf("got %d cpu and %d gpu points", len(c.CPU), len(c.GPU))
	}
}

func TestParseDefaultsInterval(t *testing.T) {
	c, err := Parse([]byte(`{"cpu":[{"temp":40,"boost":0},{"temp":90,"boost":255}],
		"gpu":[{"temp":40,"boost":0},{"temp":90,"boost":255}]}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Interval != 2 {
		t.Fatalf("interval: got %d, want 2", c.Interval)
	}
}

func TestParseRejectsGarbage(t *testing.T) {
	if _, err := Parse([]byte("not json")); !errors.Is(err, ErrInvalidCurve) {
		t.Fatalf("want ErrInvalidCurve, got %v", err)
	}
}
