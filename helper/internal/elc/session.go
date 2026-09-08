package elc

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/chr0nzz/omarchy-alienware/helper/internal/hidraw"
)

const DeviceEnv = "ALIENWARECTL_AW_ELC_DEVICE"

const resetSettleDelay = 300 * time.Millisecond

type Resetter interface {
	Reset() (string, error)
}

type Session struct {
	dev  *Device
	path string
}

func NewSession(dev *Device, path string) *Session {
	return &Session{dev: dev, path: path}
}

func (s *Session) Close() error { return s.dev.Close() }
func (s *Session) Path() string { return s.path }

func devicePath() (string, error) {
	if p := strings.TrimSpace(os.Getenv(DeviceEnv)); p != "" {
		return p, nil
	}
	info, err := hidraw.FindOneByVIDPID(DefaultVendorID, DefaultProductID)
	if err != nil {
		return "", newError(CodeNoDevice, "%s", err.Error())
	}
	return info.Path, nil
}

func open() (*Session, error) {
	path, err := devicePath()
	if err != nil {
		return nil, err
	}
	dev, err := Open(path, WriteModeOutput)
	if err != nil {
		return nil, err
	}
	return &Session{dev: dev, path: path}, nil
}

func (s *Session) looksWedged() (bool, error) {
	st, err := s.dev.Status()
	if err != nil {
		return false, err
	}
	if len(st.Raw) == 0 {
		return true, nil
	}
	for _, b := range st.Raw {
		if b != 0 {
			return false, nil
		}
	}
	return true, nil
}

func OpenWithReset(resetter Resetter) (*Session, error) {
	return openWithReset(open, resetter)
}

func openWithReset(opener func() (*Session, error), resetter Resetter) (*Session, error) {
	session, err := opener()
	if err != nil {
		return nil, err
	}
	wedged, err := session.looksWedged()
	if err != nil {
		session.Close()
		return nil, err
	}
	if !wedged {
		return session, nil
	}
	session.Close()
	if resetter == nil {
		return nil, newError(CodeWedged, "the AW-ELC controller reports an all-zero status and no reset is configured")
	}
	node, rerr := resetter.Reset()
	if rerr != nil {
		return nil, newError(CodeWedged, "the AW-ELC controller reports an all-zero status, and the reset failed: %s", rerr.Error())
	}
	time.Sleep(resetSettleDelay)
	retried, err := opener()
	if err != nil {
		return nil, err
	}
	wedged, err = retried.looksWedged()
	if err != nil {
		retried.Close()
		return nil, err
	}
	if !wedged {
		return retried, nil
	}
	retried.Close()
	return nil, newError(CodeWedged, "the AW-ELC controller at %s still reports an all-zero status after a reset", node)
}

type DeviceStatus struct {
	Path      string `json:"path"`
	VendorID  string `json:"vendorId"`
	ProductID string `json:"productId"`
	Ready     bool   `json:"ready"`
}

type RegionStatus struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	LEDCount int    `json:"ledCount"`
}

func (s *Session) Status() (DeviceStatus, []RegionStatus) {
	device := DeviceStatus{
		Path:      s.path,
		VendorID:  fmt.Sprintf("0x%04x", DefaultVendorID),
		ProductID: fmt.Sprintf("0x%04x", DefaultProductID),
		Ready:     true,
	}
	regions := make([]RegionStatus, len(Regions))
	for i, r := range Regions {
		regions[i] = RegionStatus{ID: string(r.ID), Name: r.Name, LEDCount: len(r.ZoneIDs)}
	}
	return device, regions
}

func (s *Session) SetRegion(region Region, r, g, b uint8) error {
	return s.dev.SetZoneColor(r, g, b, region.ZoneIDs)
}

func (s *Session) SetAll(r, g, b uint8) error {
	return s.dev.SetZoneColor(r, g, b, AllZoneIDs())
}

type RegionColor struct {
	Region  Region
	R, G, B uint8
}

func (s *Session) SetMap(entries []RegionColor) error {
	if len(entries) == 0 {
		return newError(CodeBadRequest, "set-map needs at least one region")
	}
	groups := make([]ColorZones, len(entries))
	for i, e := range entries {
		groups[i] = ColorZones{R: e.R, G: e.G, B: e.B, ZoneIDs: e.Region.ZoneIDs}
	}
	return s.dev.SetColorGroups(groups)
}

func (s *Session) Off() error {
	return s.dev.SetZoneColor(0, 0, 0, AllZoneIDs())
}

func (s *Session) SetBrightness(percent int) error {
	if percent < 0 || percent > 100 {
		return newError(CodeBadRequest, "brightness %d is out of range 0-100", percent)
	}
	return s.dev.Dim(percent, AllZoneIDs())
}

func (s *Session) Identify(region Region, blinks int, period time.Duration) error {
	for i := 0; i < blinks; i++ {
		if err := s.dev.SetZoneColor(255, 0, 0, region.ZoneIDs); err != nil {
			return err
		}
		time.Sleep(period)
		if err := s.dev.SetZoneColor(0, 0, 0, region.ZoneIDs); err != nil {
			return err
		}
		time.Sleep(period)
	}
	return nil
}
