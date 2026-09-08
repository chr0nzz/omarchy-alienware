package elc

import "github.com/chr0nzz/omarchy-alienware/alienfx-spike/internal/hidraw"

type Transport interface {
	SetOutputReport(buf []byte) (int, error)
	Write(buf []byte) (int, error)
	GetInput(buf []byte) (int, error)
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

func (t *hidrawTransport) SetOutputReport(buf []byte) (int, error) { return t.dev.SetOutputReport(buf) }
func (t *hidrawTransport) Write(buf []byte) (int, error)           { return t.dev.Write(buf) }
func (t *hidrawTransport) GetInput(buf []byte) (int, error)        { return t.dev.GetInput(buf) }
func (t *hidrawTransport) SetFeature(buf []byte) (int, error)      { return t.dev.SetFeature(buf) }
func (t *hidrawTransport) GetFeature(buf []byte) (int, error)      { return t.dev.GetFeature(buf) }
func (t *hidrawTransport) RawInfo() (hidraw.RawInfo, error)        { return t.dev.RawInfo() }
func (t *hidrawTransport) Close() error                            { return t.dev.Close() }
func (t *hidrawTransport) Path() string                            { return t.dev.Path() }
