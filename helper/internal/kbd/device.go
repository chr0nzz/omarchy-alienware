package kbd

import "github.com/chr0nzz/omarchy-alienware/helper/internal/hidraw"

const (
	DefaultVendorID  uint16 = 0x0d62
	DefaultProductID uint16 = 0xbabc

	DefaultKeyFirst = 0
	DefaultKeyLast  = 0x87
)

type Device struct {
	t Transport
}

func Open(path string) (*Device, error) {
	dev, err := hidraw.Open(path)
	if err != nil {
		if isBusy(err) {
			return nil, newError(CodeBusy, "kbd: %s is in use by another process: %s", path, err.Error())
		}
		return nil, newError(CodeNoDevice, "kbd: cannot open %s: %s", path, err.Error())
	}
	return &Device{t: newHidrawTransport(dev)}, nil
}

func isBusy(err error) bool {
	herr, ok := err.(*hidraw.Error)
	return ok && herr.Code() == hidraw.CodeBusy
}

func NewWithTransport(t Transport) *Device {
	return &Device{t: t}
}

func (d *Device) Close() error { return d.t.Close() }
func (d *Device) Path() string { return d.t.Path() }

func (d *Device) RawInfo() (hidraw.RawInfo, error) {
	info, err := d.t.RawInfo()
	if err != nil {
		return hidraw.RawInfo{}, newError(CodeInternal, "kbd: read raw info failed: %s", err.Error())
	}
	return info, nil
}

func (d *Device) sendFeature(frame []byte) error {
	if _, err := d.t.SetFeature(frame); err != nil {
		return newError(CodeInternal, "kbd: write failed: %s", err.Error())
	}
	return nil
}

func (d *Device) Reset() error {
	return d.sendFeature(ResetFrame())
}

type StatusReport struct {
	Raw    []byte
	Status uint8
	Name   string
	Ready  bool
}

func (d *Device) Status() (StatusReport, error) {
	if err := d.sendFeature(StatusFrame()); err != nil {
		return StatusReport{}, err
	}
	buf := make([]byte, ReportLength)
	buf[0] = FeatureReportID
	n, err := d.t.GetFeature(buf)
	if err != nil {
		return StatusReport{}, newError(CodeInternal, "kbd: read status failed: %s", err.Error())
	}
	raw := buf[:n]
	var status uint8
	if len(raw) > 2 {
		status = raw[2]
	}
	return StatusReport{Raw: raw, Status: status, Name: StatusName(status), Ready: status != StatusWaitUpdate}, nil
}

func (d *Device) SetKeyColors(keys []KeyColor) error {
	if err := d.Reset(); err != nil {
		return err
	}
	for start := 0; start < len(keys); start += MaxKeysPerColorSetFrame {
		end := start + MaxKeysPerColorSetFrame
		if end > len(keys) {
			end = len(keys)
		}
		if err := d.sendFeature(ColorSetFrame(keys[start:end])); err != nil {
			return newError(CodeInternal, "kbd: color set frame for keys %d-%d: %s", start, end-1, err.Error())
		}
	}
	if err := d.sendFeature(LoopFrame()); err != nil {
		return err
	}
	return d.sendFeature(UpdateFrame())
}

func (d *Device) SetAllKeys(r, g, b uint8, first, last int) error {
	if last < first {
		first, last = last, first
	}
	keys := make([]KeyColor, 0, last-first+1)
	for i := first; i <= last; i++ {
		keys = append(keys, KeyColor{Index: uint8(i), R: r, G: g, B: b})
	}
	return d.SetKeyColors(keys)
}

func (d *Device) SetBrightness(brightness uint8) error {
	if err := d.Reset(); err != nil {
		return err
	}
	return d.sendFeature(TurnOnFrame(brightness))
}
