package openrgb

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"time"
)

const (
	DefaultAddr   = "127.0.0.1:6742"
	ClientName    = "alienwarectl"
	dialTimeout   = 2 * time.Second
	ioTimeout     = 5 * time.Second
	versionWait   = 1 * time.Second
	maxPushSkips  = 32
	unknownServer = uint32(0)
)

const (
	CodeNoOpenRGB    = "no-openrgb"
	CodeBadRequest   = "bad-request"
	CodeNotSupported = "not-supported"
	CodeInternal     = "internal"
)

type Error struct {
	code string
	msg  string
}

func (e *Error) Error() string { return e.msg }
func (e *Error) Code() string  { return e.code }

func newError(code, format string, args ...any) *Error {
	return &Error{code: code, msg: fmt.Sprintf(format, args...)}
}

type Client struct {
	addr    string
	conn    net.Conn
	version uint32
}

func Dial(addr string) (*Client, error) {
	if addr == "" {
		addr = DefaultAddr
	}
	conn, err := net.DialTimeout("tcp", addr, dialTimeout)
	if err != nil {
		return nil, newError(CodeNoOpenRGB, "cannot reach the OpenRGB SDK server at %s: %s", addr, err.Error())
	}
	c := &Client{addr: addr, conn: conn, version: unknownServer}
	if err := c.handshake(); err != nil {
		conn.Close()
		return nil, err
	}
	return c, nil
}

func (c *Client) Addr() string { return c.addr }

func (c *Client) Version() uint32 { return c.version }

func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}
	err := c.conn.Close()
	c.conn = nil
	return err
}

func (c *Client) send(deviceID, packetID uint32, payload []byte) error {
	if err := c.conn.SetWriteDeadline(time.Now().Add(ioTimeout)); err != nil {
		return newError(CodeInternal, "%s", err.Error())
	}
	frame := append(EncodeHeader(Header{DeviceID: deviceID, PacketID: packetID, Size: uint32(len(payload))}), payload...)
	if _, err := c.conn.Write(frame); err != nil {
		return newError(CodeNoOpenRGB, "lost the OpenRGB connection while sending: %s", err.Error())
	}
	return nil
}

func (c *Client) readPacket(deadline time.Time) (Header, []byte, error) {
	if err := c.conn.SetReadDeadline(deadline); err != nil {
		return Header{}, nil, newError(CodeInternal, "%s", err.Error())
	}
	raw := make([]byte, HeaderSize)
	if _, err := io.ReadFull(c.conn, raw); err != nil {
		return Header{}, nil, err
	}
	h, err := DecodeHeader(raw)
	if err != nil {
		return Header{}, nil, err
	}
	body := make([]byte, h.Size)
	if h.Size > 0 {
		if _, err := io.ReadFull(c.conn, body); err != nil {
			return Header{}, nil, err
		}
	}
	return h, body, nil
}

func isPush(packetID uint32) bool {
	switch packetID {
	case PktDeviceListUpdated, PktDetectionStarted, PktDetectionProgress, PktDetectionComplete, PktSignalUpdate:
		return true
	}
	return packetID >= 150 && packetID <= 161
}

func (c *Client) await(packetID uint32, deadline time.Time) ([]byte, error) {
	for i := 0; i < maxPushSkips; i++ {
		h, body, err := c.readPacket(deadline)
		if err != nil {
			var nerr net.Error
			if errors.As(err, &nerr) && nerr.Timeout() {
				return nil, newError(CodeNoOpenRGB, "the OpenRGB SDK server did not answer packet %d in time", packetID)
			}
			if errors.Is(err, ErrMalformed) {
				return nil, newError(CodeInternal, "%s", err.Error())
			}
			return nil, newError(CodeNoOpenRGB, "lost the OpenRGB connection while waiting for packet %d: %s", packetID, err.Error())
		}
		if h.PacketID == packetID {
			return body, nil
		}
		if isPush(h.PacketID) {
			continue
		}
	}
	return nil, newError(CodeInternal, "the OpenRGB SDK server never sent packet %d", packetID)
}

func (c *Client) handshake() error {
	if err := c.send(0, PktRequestProtocolVersion, binaryU32(ClientProtocolVersion)); err != nil {
		return err
	}
	body, err := c.await(PktRequestProtocolVersion, time.Now().Add(versionWait))
	if err != nil {
		var e *Error
		if errors.As(err, &e) && e.Code() == CodeNoOpenRGB {
			c.version = unknownServer
		} else {
			return err
		}
	} else if len(body) >= 4 {
		server := binary.LittleEndian.Uint32(body[:4])
		c.version = server
		if server > ClientProtocolVersion {
			c.version = ClientProtocolVersion
		}
	}
	name := append([]byte(ClientName), 0)
	return c.send(0, PktSetClientName, name)
}

func binaryU32(v uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	return b
}

func (c *Client) ControllerCount() (int, error) {
	if err := c.send(0, PktRequestControllerCount, nil); err != nil {
		return 0, err
	}
	body, err := c.await(PktRequestControllerCount, time.Now().Add(ioTimeout))
	if err != nil {
		return 0, err
	}
	if len(body) < 4 {
		return 0, newError(CodeInternal, "the controller count reply was %d bytes, want at least 4", len(body))
	}
	return int(binary.LittleEndian.Uint32(body[:4])), nil
}

func (c *Client) Controller(index int) (Controller, error) {
	var payload []byte
	if c.version > 0 {
		payload = binaryU32(c.version)
	}
	if err := c.send(uint32(index), PktRequestControllerData, payload); err != nil {
		return Controller{}, err
	}
	body, err := c.await(PktRequestControllerData, time.Now().Add(ioTimeout))
	if err != nil {
		return Controller{}, err
	}
	ctrl, err := DecodeControllerData(body, c.version)
	if err != nil {
		return Controller{}, newError(CodeInternal, "%s", err.Error())
	}
	ctrl.Index = index
	return ctrl, nil
}

func (c *Client) ControllerRaw(index int) ([]byte, error) {
	var payload []byte
	if c.version > 0 {
		payload = binaryU32(c.version)
	}
	if err := c.send(uint32(index), PktRequestControllerData, payload); err != nil {
		return nil, err
	}
	return c.await(PktRequestControllerData, time.Now().Add(ioTimeout))
}

func (c *Client) SetClientName(name string) error {
	return c.send(0, PktSetClientName, append([]byte(name), 0))
}

func (c *Client) UpdateZoneLEDs(deviceIndex int, zoneIndex uint32, colors []uint32) error {
	return c.send(uint32(deviceIndex), PktUpdateZoneLEDs, EncodeUpdateZoneLEDs(zoneIndex, colors))
}

func (c *Client) UpdateLEDs(deviceIndex int, colors []uint32) error {
	return c.send(uint32(deviceIndex), PktUpdateLEDs, EncodeUpdateLEDs(colors))
}

func (c *Client) UpdateMode(deviceIndex int, modeIndex int32, m Mode) error {
	return c.send(uint32(deviceIndex), PktUpdateMode, EncodeUpdateMode(modeIndex, m, c.version))
}
