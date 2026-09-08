package fan

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
)

const (
	MinTemp       = 0
	MaxTemp       = 110
	MinBoost      = 0
	MaxBoost      = 255
	MinInterval   = 1
	MaxInterval   = 30
	MinHysteresis = 0
	MaxHysteresis = 15
	MinPoints     = 2
	MaxPoints     = 8
)

var ErrInvalidCurve = errors.New("invalid curve")

type Point struct {
	Temp  int `json:"temp"`
	Boost int `json:"boost"`
}

type Curve struct {
	Interval   int     `json:"interval"`
	Hysteresis int     `json:"hysteresis"`
	CPU        []Point `json:"cpu"`
	GPU        []Point `json:"gpu"`
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func normalizePoints(in []Point) []Point {
	out := make([]Point, 0, len(in))
	for _, p := range in {
		out = append(out, Point{
			Temp:  clampInt(p.Temp, MinTemp, MaxTemp),
			Boost: clampInt(p.Boost, MinBoost, MaxBoost),
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Temp < out[j].Temp })
	dedup := make([]Point, 0, len(out))
	for _, p := range out {
		if n := len(dedup); n > 0 && dedup[n-1].Temp == p.Temp {
			if p.Boost > dedup[n-1].Boost {
				dedup[n-1].Boost = p.Boost
			}
			continue
		}
		dedup = append(dedup, p)
	}
	return dedup
}

func validateSide(name string, in []Point) ([]Point, error) {
	if len(in) > MaxPoints {
		return nil, fmt.Errorf("%w: %s has %d points, maximum is %d", ErrInvalidCurve, name, len(in), MaxPoints)
	}
	out := normalizePoints(in)
	if len(out) < MinPoints {
		return nil, fmt.Errorf("%w: %s needs at least %d distinct points, got %d", ErrInvalidCurve, name, MinPoints, len(out))
	}
	return out, nil
}

func Normalize(c Curve) (Curve, error) {
	cpu, err := validateSide("cpu", c.CPU)
	if err != nil {
		return Curve{}, err
	}
	gpu, err := validateSide("gpu", c.GPU)
	if err != nil {
		return Curve{}, err
	}
	return Curve{
		Interval:   clampInt(c.Interval, MinInterval, MaxInterval),
		Hysteresis: clampInt(c.Hysteresis, MinHysteresis, MaxHysteresis),
		CPU:        cpu,
		GPU:        gpu,
	}, nil
}

func Parse(data []byte) (Curve, error) {
	var raw Curve
	if err := json.Unmarshal(data, &raw); err != nil {
		return Curve{}, fmt.Errorf("%w: %s", ErrInvalidCurve, err.Error())
	}
	if raw.Interval == 0 {
		raw.Interval = 2
	}
	return Normalize(raw)
}

func Interpolate(points []Point, temp float64) int {
	if len(points) == 0 {
		return 0
	}
	if temp <= float64(points[0].Temp) {
		return points[0].Boost
	}
	last := len(points) - 1
	if temp >= float64(points[last].Temp) {
		return points[last].Boost
	}
	for i := 0; i < last; i++ {
		a := points[i]
		b := points[i+1]
		if temp < float64(a.Temp) || temp > float64(b.Temp) {
			continue
		}
		span := float64(b.Temp - a.Temp)
		if span <= 0 {
			return b.Boost
		}
		ratio := (temp - float64(a.Temp)) / span
		v := float64(a.Boost) + ratio*float64(b.Boost-a.Boost)
		return clampInt(int(math.Round(v)), MinBoost, MaxBoost)
	}
	return points[last].Boost
}
