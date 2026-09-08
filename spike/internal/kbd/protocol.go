package kbd

const (
	ReportLength          = 64
	PayloadLength         = ReportLength - 1
	FeatureReportID uint8 = 0xcc

	colorBlockStart = 4
	colorBlockSize  = 4

	MaxKeysPerColorSetFrame = (ReportLength - colorBlockStart) / colorBlockSize
)

const (
	opReset    uint8 = 0x94
	opStatus   uint8 = 0x93
	opColor    uint8 = 0x8c
	subSet     uint8 = 0x02
	subLoop    uint8 = 0x13
	opUpdate   uint8 = 0x8b
	subUpdate  uint8 = 0x01
	argUpdate  uint8 = 0xff
	opTurnOn   uint8 = 0x83
	subTurnOn1 uint8 = 0x38
	subTurnOn2 uint8 = 0x9c

	turnOnBrightnessOffset = 4
)

const (
	StatusWaitUpdate   uint8 = 0x80
	StatusStartCommand uint8 = 0x8c
	StatusInCommand    uint8 = 0xcc
)

func StatusName(status uint8) string {
	switch status {
	case 0:
		return "zero"
	case StatusWaitUpdate:
		return "wait-update"
	case StatusStartCommand:
		return "start-command"
	case StatusInCommand:
		return "in-command"
	default:
		return "unknown"
	}
}

func newFrame() []byte {
	b := make([]byte, ReportLength)
	b[0] = FeatureReportID
	return b
}

func ResetFrame() []byte {
	b := newFrame()
	b[1] = opReset
	return b
}

func StatusFrame() []byte {
	b := newFrame()
	b[1] = opStatus
	return b
}

type KeyColor struct {
	Index   uint8
	R, G, B uint8
}

func ColorSetFrame(keys []KeyColor) []byte {
	b := newFrame()
	b[1] = opColor
	b[2] = subSet
	pos := colorBlockStart
	for _, k := range keys {
		if pos+colorBlockSize > ReportLength {
			break
		}
		b[pos] = k.Index + 1
		b[pos+1] = k.R
		b[pos+2] = k.G
		b[pos+3] = k.B
		pos += colorBlockSize
	}
	return b
}

func LoopFrame() []byte {
	b := newFrame()
	b[1] = opColor
	b[2] = subLoop
	return b
}

func UpdateFrame() []byte {
	b := newFrame()
	b[1] = opUpdate
	b[2] = subUpdate
	b[3] = argUpdate
	return b
}

func TurnOnFrame(brightness uint8) []byte {
	b := newFrame()
	b[1] = opTurnOn
	b[2] = subTurnOn1
	b[3] = subTurnOn2
	b[turnOnBrightnessOffset] = brightness
	return b
}
