package internal

type Opcode uint8

const (
	OpcodeContinuationFrame Opcode = iota
	OpcodeTextFrame
	OpcodeBinaryFrame
	OpcodeNonControlFrame1
	OpcodeNonControlFrame2
	OpcodeNonControlFrame3
	OpcodeNonControlFrame4
	OpcodeNonControlFrame5
	OpcodeConnectionClose
	OpcodePing
	OpcodePong
	OpcodeControlFrame1
	OpcodeControlFrame2
	OpcodeControlFrame3
	OpcodeControlFrame4
	OpcodeControlFrame5
)

func (c Opcode) IsControl() bool {
	if c == OpcodeConnectionClose ||
		c == OpcodePing ||
		c == OpcodePong {
		return true
	} else {
		return false
	}
}

func (c Opcode) IsData() bool {
	if c == OpcodeContinuationFrame ||
		c == OpcodeTextFrame ||
		c == OpcodeBinaryFrame {
		return true
	} else {
		return false
	}
}

func (c Opcode) IsReserved() bool {
	return !c.IsControl() && !c.IsData()
}
