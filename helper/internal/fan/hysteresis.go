package fan

type State struct {
	hysteresis float64
	primed     bool
	boost      int
	heldTemp   float64
}

func NewState(hysteresis int) *State {
	return &State{hysteresis: float64(clampInt(hysteresis, MinHysteresis, MaxHysteresis))}
}

func (s *State) Boost() int {
	return s.boost
}

func (s *State) HeldTemp() float64 {
	return s.heldTemp
}

func (s *State) Reset() {
	s.primed = false
	s.boost = 0
	s.heldTemp = 0
}

func (s *State) Update(points []Point, temp float64) int {
	target := Interpolate(points, temp)
	if !s.primed {
		s.primed = true
		s.boost = target
		s.heldTemp = temp
		return s.boost
	}
	if target >= s.boost {
		s.boost = target
		s.heldTemp = temp
		return s.boost
	}
	if s.heldTemp-temp > s.hysteresis {
		s.boost = target
		s.heldTemp = temp
		return s.boost
	}
	return s.boost
}
