package elc

import (
	"time"

	"github.com/chr0nzz/omarchy-alienware/helper/internal/hidraw"
)

type WriteMode int

const (
	WriteModeOutput WriteMode = iota
	WriteModeWrite
	WriteModeFeature
)

func (m WriteMode) String() string {
	switch m {
	case WriteModeOutput:
		return "output"
	case WriteModeWrite:
		return "write"
	case WriteModeFeature:
		return "feature"
	default:
		return "unknown"
	}
}

func ParseWriteMode(s string) (WriteMode, error) {
	switch s {
	case "", "output":
		return WriteModeOutput, nil
	case "write":
		return WriteModeWrite, nil
	case "feature":
		return WriteModeFeature, nil
	default:
		return 0, newError(CodeBadRequest, "elc: unknown transport %q, want output, write or feature", s)
	}
}

const (
	DefaultVendorID  uint16 = 0x187c
	DefaultProductID uint16 = 0x0550
)

var ErrDeviceTimeout = newError(CodeTimeout, "elc: device did not report ready before the timeout")

type Device struct {
	t         Transport
	writeMode WriteMode
}

func Open(path string, writeMode WriteMode) (*Device, error) {
	dev, err := hidraw.Open(path)
	if err != nil {
		return nil, newError(CodeNoDevice, "elc: cannot open %s: %s", path, err.Error())
	}
	return &Device{t: newHidrawTransport(dev), writeMode: writeMode}, nil
}

func NewWithTransport(t Transport, writeMode WriteMode) *Device {
	return &Device{t: t, writeMode: writeMode}
}

func (d *Device) Close() error { return d.t.Close() }
func (d *Device) Path() string { return d.t.Path() }

func (d *Device) send(frame []byte) error {
	var err error
	switch d.writeMode {
	case WriteModeOutput:
		_, err = d.t.SetOutputReport(frame)
	case WriteModeWrite:
		_, err = d.t.Write(frame)
	case WriteModeFeature:
		_, err = d.t.SetFeature(frame)
	default:
		return newError(CodeInternal, "elc: unknown write mode %v", d.writeMode)
	}
	if err != nil {
		return newError(CodeInternal, "elc: write failed: %s", err.Error())
	}
	return nil
}

type StatusReport struct {
	Raw    []byte
	Status uint8
	Name   string
}

func (d *Device) Status() (StatusReport, error) {
	buf := make([]byte, ReportLength)
	n, err := d.t.GetInput(buf)
	if err != nil {
		return StatusReport{}, newError(CodeInternal, "elc: read status failed: %s", err.Error())
	}
	raw := buf[:n]
	var status uint8
	if len(raw) > 2 {
		status = raw[2]
	}
	return StatusReport{Raw: raw, Status: status, Name: StatusName(status)}, nil
}

func (d *Device) FeatureProbe() ([]byte, error) {
	buf := make([]byte, ReportLength)
	n, err := d.t.GetFeature(buf)
	if err != nil {
		return nil, newError(CodeInternal, "elc: read feature report failed: %s", err.Error())
	}
	return buf[:n], nil
}

func (d *Device) WaitForReady(timeout time.Duration) (StatusReport, error) {
	deadline := time.Now().Add(timeout)
	var last StatusReport
	for {
		st, err := d.Status()
		if err != nil {
			return StatusReport{}, err
		}
		last = st
		if st.Status == 0 || st.Status != StatusV4Busy {
			return st, nil
		}
		if time.Now().After(deadline) {
			return last, ErrDeviceTimeout
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func (d *Device) Reset() error {
	if _, err := d.WaitForReady(2 * time.Second); err != nil {
		return err
	}
	if err := d.send(ControlFrame(ControlRemove, ControlIDCommon)); err != nil {
		return err
	}
	return d.send(ControlFrame(ControlStartNew, ControlIDCommon))
}

func (d *Device) Apply() error {
	return d.send(ControlFrame(ControlFinishPlay, ControlIDCommon))
}

func (d *Device) SelectZones(loop bool, zoneIDs []uint8) error {
	if len(zoneIDs) > MaxSelectZonesPerFrame {
		return newError(CodeBadRequest, "elc: %d zones exceeds the %d a single select-zones frame can carry", len(zoneIDs), MaxSelectZonesPerFrame)
	}
	return d.send(SelectZonesFrame(loop, zoneIDs))
}

func (d *Device) AddAction(phases []Phase) error {
	if len(phases) > MaxActionPhasesPerFrame {
		return newError(CodeBadRequest, "elc: %d phases exceeds the %d a single add-action frame can carry", len(phases), MaxActionPhasesPerFrame)
	}
	return d.send(AddActionFrame(phases))
}

func (d *Device) SetColorZones(r, g, b uint8, zoneIDs []uint8) error {
	if len(zoneIDs) == 0 {
		return d.send(SetOneColorFrame(r, g, b, nil))
	}
	for start := 0; start < len(zoneIDs); start += MaxColorZonesPerFrame {
		end := start + MaxColorZonesPerFrame
		if end > len(zoneIDs) {
			end = len(zoneIDs)
		}
		if err := d.send(SetOneColorFrame(r, g, b, zoneIDs[start:end])); err != nil {
			return err
		}
	}
	return nil
}

func (d *Device) Dim(percent int, zoneIDs []uint8) error {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	dim := uint8(100 - percent)
	for start := 0; start < len(zoneIDs); start += MaxTurnOnZonesPerFrame {
		end := start + MaxTurnOnZonesPerFrame
		if end > len(zoneIDs) {
			end = len(zoneIDs)
		}
		if err := d.send(TurnOnFrame(dim, zoneIDs[start:end])); err != nil {
			return err
		}
	}
	return nil
}

type ColorZones struct {
	R, G, B uint8
	ZoneIDs []uint8
}

func (d *Device) SetColorGroups(groups []ColorZones) error {
	if err := d.Reset(); err != nil {
		return err
	}
	for _, group := range groups {
		if err := d.SetColorZones(group.R, group.G, group.B, group.ZoneIDs); err != nil {
			return err
		}
	}
	return d.Apply()
}

func (d *Device) SetZoneColor(r, g, b uint8, zoneIDs []uint8) error {
	return d.SetColorGroups([]ColorZones{{R: r, G: g, B: b, ZoneIDs: zoneIDs}})
}

func (d *Device) RawInfo() (hidraw.RawInfo, error) {
	info, err := d.t.RawInfo()
	if err != nil {
		return hidraw.RawInfo{}, newError(CodeInternal, "elc: read raw info failed: %s", err.Error())
	}
	return info, nil
}
