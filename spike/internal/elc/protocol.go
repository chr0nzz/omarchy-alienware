package elc

const (
	ReportLength  = 34
	PayloadLength = 33

	MaxColorZonesPerFrame   = ReportLength - 8
	MaxSelectZonesPerFrame  = ReportLength - 6
	MaxTurnOnZonesPerFrame  = ReportLength - 6
	MaxActionPhasesPerFrame = 3
)

const (
	opControl     = 0x21
	opSetPower    = 0x22
	opSelectZones = 0x23
	opAddAction   = 0x24
	opTurnOn      = 0x26
	opSetOneColor = 0x27

	subsystemByte = 0x03
)

const (
	ControlStartNew   uint8 = 1
	ControlFinishSave uint8 = 2
	ControlFinishPlay uint8 = 3
	ControlRemove     uint8 = 4
	ControlPlay       uint8 = 5
	ControlSetDefault uint8 = 6
	ControlSetStartup uint8 = 7
)

const (
	ControlIDCommon  uint16 = 0xffff
	ControlIDStartup uint16 = 0x0008
	ControlIDLight   uint16 = 0x0061
)

type ActionType uint8

const (
	ActionColor     ActionType = 0
	ActionPulse     ActionType = 1
	ActionMorph     ActionType = 2
	ActionBreathing ActionType = 3
	ActionSpectrum  ActionType = 4
	ActionRainbow   ActionType = 5
	ActionPower     ActionType = 6
)

var actionOpCodes = [7]uint8{0xd0, 0xdc, 0xcf, 0xdc, 0x82, 0xac, 0xe8}

type Phase struct {
	Type    ActionType
	Time    uint8
	Tempo   uint8
	R, G, B uint8
}

func newFrame() []byte {
	return make([]byte, ReportLength)
}

func ControlFrame(controlType uint8, controlID uint16) []byte {
	b := newFrame()
	b[1] = subsystemByte
	b[2] = opControl
	b[4] = controlType
	b[5] = uint8(controlID >> 8)
	b[6] = uint8(controlID)
	return b
}

func SelectZonesFrame(loop bool, zoneIDs []uint8) []byte {
	b := newFrame()
	b[1] = subsystemByte
	b[2] = opSelectZones
	if loop {
		b[3] = 1
	}
	b[5] = uint8(len(zoneIDs))
	copy(b[6:], zoneIDs)
	return b
}

func AddActionFrame(phases []Phase) []byte {
	b := newFrame()
	b[1] = subsystemByte
	b[2] = opAddAction
	pos := 3
	for i, p := range phases {
		if i >= MaxActionPhasesPerFrame {
			break
		}
		frameType := p.Type
		if frameType >= ActionBreathing {
			frameType = ActionMorph
		}
		tempo := p.Tempo
		if p.Type == ActionColor {
			tempo = 0xfa
		}
		b[pos] = uint8(frameType)
		b[pos+1] = p.Time
		b[pos+2] = actionOpCodes[p.Type]
		b[pos+4] = tempo
		b[pos+5] = p.R
		b[pos+6] = p.G
		b[pos+7] = p.B
		pos += 8
	}
	return b
}

func SetOneColorFrame(r, g, b uint8, zoneIDs []uint8) []byte {
	buf := newFrame()
	buf[1] = subsystemByte
	buf[2] = opSetOneColor
	buf[3] = r
	buf[4] = g
	buf[5] = b
	buf[7] = uint8(len(zoneIDs))
	copy(buf[8:], zoneIDs)
	return buf
}

func TurnOnFrame(dim uint8, zoneIDs []uint8) []byte {
	buf := newFrame()
	buf[1] = subsystemByte
	buf[2] = opTurnOn
	buf[3] = dim
	buf[5] = uint8(len(zoneIDs))
	copy(buf[6:], zoneIDs)
	return buf
}

func SetPowerControlFrame(controlType uint8, powerID uint8) []byte {
	b := newFrame()
	b[1] = subsystemByte
	b[2] = opSetPower
	b[4] = controlType
	b[6] = powerID
	return b
}

const (
	StatusV4Ready      uint8 = 33
	StatusV4Busy       uint8 = 34
	StatusV4WaitColor  uint8 = 35
	StatusV4WaitUpdate uint8 = 36
	StatusV4WasOn      uint8 = 38
)

func StatusName(status uint8) string {
	switch status {
	case StatusV4Ready:
		return "ready"
	case StatusV4Busy:
		return "busy"
	case StatusV4WaitColor:
		return "wait-color"
	case StatusV4WaitUpdate:
		return "wait-update"
	case StatusV4WasOn:
		return "was-on"
	case 0:
		return "zero"
	default:
		return "unknown"
	}
}
