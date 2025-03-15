package internal

type FrameHeader struct {
	IsFinalFrame bool
	// 1 bit
	RSV1 uint8
	// 1 bit
	RSV2 uint8
	// 1 bit
	RSV3 uint8
	// 4 bits
	Opcode Opcode
	// 1 bit
	IsMasked bool
	// 7 bits, 7+16 bits, or 7+64 bits
	PayloadLength uint64
	// 0 or 4 bytes
	MaskingKey [4]byte
	// 0 bytes unless an extension has been negotiated
	ExtensionData []byte
}
