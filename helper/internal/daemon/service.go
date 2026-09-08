package daemon

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"

	"github.com/chr0nzz/omarchy-alienware/helper/internal/fan"
	"github.com/chr0nzz/omarchy-alienware/helper/internal/hw"
	"github.com/chr0nzz/omarchy-alienware/helper/internal/kbd"
)

const customProfile = "custom"

type Service struct {
	reader *hw.Reader
	auth   authorizer
	logger *log.Logger

	ctrl sync.Mutex

	mu           sync.Mutex
	active       bool
	last         fan.Curve
	cancel       context.CancelFunc
	done         chan struct{}
	savedProfile string

	kbdMu           sync.Mutex
	openKeyboard    func() (*kbd.Device, error)
	keyboardPresent func() bool
}

func NewService(reader *hw.Reader, auth authorizer, logger *log.Logger) *Service {
	if auth == nil {
		auth = allowAll{}
	}
	if logger == nil {
		logger = log.Default()
	}
	return &Service{
		reader:          reader,
		auth:            auth,
		logger:          logger,
		last:            fan.Curve{Interval: 2, Hysteresis: 3, CPU: []fan.Point{}, GPU: []fan.Point{}},
		openKeyboard:    kbd.OpenDefault,
		keyboardPresent: kbd.Present,
	}
}

func (s *Service) curveStatus() CurveStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return CurveStatus{
		Active:     s.active,
		Interval:   s.last.Interval,
		Hysteresis: s.last.Hysteresis,
		CPU:        s.last.CPU,
		GPU:        s.last.GPU,
	}
}

func (s *Service) Status() (string, *dbus.Error) {
	out, err := BuildStatus(s.reader.Snapshot(), s.curveStatus()).JSON()
	if err != nil {
		return "", mapError(err)
	}
	return out, nil
}

func (s *Service) SetProfile(name string, sender dbus.Sender) *dbus.Error {
	if err := s.auth.Authorize(sender, ActionSetProfile); err != nil {
		return mapError(err)
	}
	return mapError(s.reader.SetProfile(name))
}

func (s *Service) SetBoost(fanID string, value uint32, sender dbus.Sender) *dbus.Error {
	if err := s.auth.Authorize(sender, ActionSetFan); err != nil {
		return mapError(err)
	}
	if value > 255 {
		return dbusError(ErrNameBadRequest, "boost must be 0-255")
	}
	s.mu.Lock()
	active := s.active
	s.mu.Unlock()
	if active {
		return dbusError(ErrNameBadRequest, "a fan curve is active, stop it before setting boost manually")
	}
	return mapError(s.reader.SetBoost(fanID, int(value)))
}

func (s *Service) ApplyCurve(payload string, sender dbus.Sender) *dbus.Error {
	if err := s.auth.Authorize(sender, ActionSetFan); err != nil {
		return mapError(err)
	}
	if err := s.auth.Authorize(sender, ActionSetProfile); err != nil {
		return mapError(err)
	}
	curve, err := fan.Parse([]byte(payload))
	if err != nil {
		return mapError(err)
	}
	return mapError(s.StartCurve(curve))
}

func (s *Service) StopCurve(sender dbus.Sender) *dbus.Error {
	s.Stop()
	return nil
}

func (s *Service) SetTurbo(on bool, sender dbus.Sender) *dbus.Error {
	if err := s.auth.Authorize(sender, ActionSetPower); err != nil {
		return mapError(err)
	}
	return mapError(s.reader.SetTurbo(on))
}

func (s *Service) SetPowerLimit(constraint uint32, watts uint32, sender dbus.Sender) *dbus.Error {
	if err := s.auth.Authorize(sender, ActionSetPower); err != nil {
		return mapError(err)
	}
	return mapError(s.reader.SetPowerLimit(int(constraint), int(watts)))
}

func (s *Service) StartCurve(curve fan.Curve) error {
	s.ctrl.Lock()
	defer s.ctrl.Unlock()
	return s.startCurveLocked(curve)
}

func (s *Service) startCurveLocked(curve fan.Curve) error {
	s.stopLocked()

	saved, err := s.reader.ReadProfile()
	if err != nil {
		saved = ""
		s.logger.Printf("curve: current platform profile unreadable, will not restore one on stop")
	}
	if err := s.reader.SetProfile(customProfile); err != nil {
		s.logger.Printf("curve: could not select the %s platform profile: %v", customProfile, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	s.mu.Lock()
	s.active = true
	s.last = curve
	s.cancel = cancel
	s.done = done
	s.savedProfile = saved
	s.mu.Unlock()

	go s.run(ctx, curve, done)
	return nil
}

func (s *Service) Stop() {
	s.ctrl.Lock()
	defer s.ctrl.Unlock()
	s.stopLocked()
}

func (s *Service) stopLocked() {
	s.mu.Lock()
	cancel := s.cancel
	done := s.done
	saved := s.savedProfile
	s.cancel = nil
	s.done = nil
	s.savedProfile = ""
	wasActive := s.active
	s.active = false
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
	if err := s.reader.ResetBoost(); err != nil {
		s.logger.Printf("failsafe: could not reset fan boost: %v", err)
	}
	if wasActive && saved != "" && saved != customProfile {
		current, err := s.reader.ReadProfile()
		if err == nil && current != customProfile {
			s.logger.Printf("curve: leaving the %s platform profile alone, it was changed while the curve ran", current)
			return
		}
		if err := s.reader.SetProfile(saved); err != nil {
			s.logger.Printf("curve: could not restore the %s platform profile: %v", saved, err)
		}
	}
}

func (s *Service) run(ctx context.Context, curve fan.Curve, done chan struct{}) {
	defer close(done)
	defer func() {
		if err := s.reader.ResetBoost(); err != nil {
			s.logger.Printf("failsafe: could not reset fan boost when the curve loop stopped: %v", err)
		}
	}()
	defer func() {
		if rec := recover(); rec != nil {
			s.logger.Printf("curve loop panicked: %v", rec)
			s.mu.Lock()
			if s.done == done {
				s.active = false
				s.cancel = nil
				s.done = nil
				s.savedProfile = ""
			}
			s.mu.Unlock()
		}
	}()

	states := map[string]*fan.State{
		hw.FanCPU: fan.NewState(curve.Hysteresis),
		hw.FanGPU: fan.NewState(curve.Hysteresis),
	}
	points := map[string][]fan.Point{
		hw.FanCPU: curve.CPU,
		hw.FanGPU: curve.GPU,
	}

	ticker := time.NewTicker(time.Duration(curve.Interval) * time.Second)
	defer ticker.Stop()

	for {
		s.tick(states, points)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Service) tick(states map[string]*fan.State, points map[string][]fan.Point) {
	temps, _ := s.reader.Temps()
	for _, id := range []string{hw.FanCPU, hw.FanGPU} {
		temp, ok := temps[id]
		if !ok {
			states[id].Reset()
			if err := s.reader.SetBoost(id, 0); err != nil {
				s.logger.Printf("curve: %s temperature unreadable and boost reset failed: %v", id, err)
			}
			continue
		}
		boost := states[id].Update(points[id], float64(temp))
		if err := s.reader.SetBoost(id, boost); err != nil {
			s.logger.Printf("curve: could not set %s boost: %v", id, err)
		}
	}
}
