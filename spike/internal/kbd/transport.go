package kbd

import "github.com/chr0nzz/omarchy-alienware/alienfx-spike/internal/hidraw"

type Transport interface {
	SetFeature(buf []byte) (int, error)
	GetFeature(buf []byte) (int, error)
	RawInfo() (hidraw.RawInfo, error)
	Close() error
	Path() string
}

type hidrawTransport struct {
	dev *hidraw.Device
}

func newHidrawTransport(dev *hidraw.Device) Transport {
	return &hidrawTransport{dev: dev}
}

func (t *hidrawTransport) SetFeature(buf []byte) (int, error) { return t.dev.SetFeature(buf) }
func (t *hidrawTransport) GetFeature(buf []byte) (int, error) { return t.dev.GetFeature(buf) }
func (t *hidrawTransport) RawInfo() (hidraw.RawInfo, error)   { return t.dev.RawInfo() }
func (t *hidrawTransport) Close() error                       { return t.dev.Close() }
func (t *hidrawTransport) Path() string                       { return t.dev.Path() }
