package frame

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

/*
  0                   1                   2                   3
  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
 +-+-+-+-+-------+-+-------------+-------------------------------+
 |F|R|R|R| opcode|M| Payload len |    Extended payload length    |
 |I|S|S|S|  (4)  |A|     (7)     |             (16/64)           |
 |N|V|V|V|       |S|             |   (if payload len==126/127)   |
 | |1|2|3|       |K|             |                               |
 +-+-+-+-+-------+-+-------------+ - - - - - - - - - - - - - - - +
 |     Extended payload length continued, if payload len == 127  |
 + - - - - - - - - - - - - - - - +-------------------------------+
 |                               |Masking-key, if MASK set to 1  |
 +-------------------------------+-------------------------------+
 | Masking-key (continued)       |          Payload Data         |
 +-------------------------------- - - - - - - - - - - - - - - - +
 :                     Payload Data continued ...                :
 + - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - +
 |                     Payload Data continued ...                |
 +---------------------------------------------------------------+
*/

type Frame struct {
	// 1 bit, is the final fragment in a message
	FIN FIN
	// 1 bit
	RSV1 uint8
	// 1 bit
	RSV2 uint8
	// 1 bit
	RSV3 uint8
	// 4 bits
	Opcode Opcode
	// 1 bit
	Mask Mask
	// 7 bits, 7+16 bits, or 7+64 bits
	PayloadLength uint64
	// 0 or 4 bytes
	MaskingKey uint32
	// 0 bytes unless an extension has been negotiated
	ExtensionData []byte
	// Arbitrary "Application data", PayloadLength - len(ExtensionData)
	ApplicationData []byte
}

type FIN uint8

const (
	FINMoreFramesToFollow FIN = iota
	FINFinalFrame
)

type Mask uint8

const (
	MaskDataNotMasked Mask = iota
	MaskDataMasked
)

func (m Mask) MaskBytesSize() uint8 {
	if m == MaskDataNotMasked {
		return 0
	}
	return 4
}

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
	if c >= OpcodeConnectionClose {
		return true
	} else {
		return false
	}
}

type PayloadLengthType uint8

const (
	// 0-125, first 7 bits represent length as is
	PayloadLengthTypeShort PayloadLengthType = iota
	// 126-2^16, first 7 bits = 126, next 16 bits represent length
	PayloadLengthTypeMedium
	// 2^16-2^64, first 7 bits = 127, next 64 bits represent length
	PayloadLengthTypeHigh
)

func payloadLengthTypeFromPayloadLength(l uint64) PayloadLengthType {
	return PayloadLengthTypeShort
}

// Close codes defined in RFC 6455, section 11.7.
const (
	CloseNormalClosure           = 1000
	CloseGoingAway               = 1001
	CloseProtocolError           = 1002
	CloseUnsupportedData         = 1003
	CloseNoStatusReceived        = 1005
	CloseAbnormalClosure         = 1006
	CloseInvalidFramePayloadData = 1007
	ClosePolicyViolation         = 1008
	CloseMessageTooBig           = 1009
	CloseMandatoryExtension      = 1010
	CloseInternalServerErr       = 1011
	CloseServiceRestart          = 1012
	CloseTryAgainLater           = 1013
	CloseTLSHandshake            = 1015
)

func readNBytes(r *bufio.Reader, n int) ([]byte, error) {
	bytes, err := r.Peek(n)
	if err == io.EOF {
		return nil, io.ErrUnexpectedEOF
	}

	r.Discard(len(bytes))

	return bytes, err
}

func ReadFrame(r *bufio.Reader) (*Frame, error) {
	h, err := readNBytes(r, 2)
	if err != nil {
		return nil, fmt.Errorf("failed to read first 2 essential bytes of the frame: [%w]", err)
	}
	b0, b1 := h[0], h[1]

	f := Frame{}
	f.FIN = FIN(b0 & 0b1_000_0000 >> 7)
	f.RSV1 = b0 & 0b0_100_0000 >> 6
	f.RSV2 = b0 & 0b0_010_0000 >> 5
	f.RSV3 = b0 & 0b0_001_0000 >> 4
	f.Opcode = Opcode(b0 & 0b0_000_1111)

	f.Mask = Mask(b1 & 0b1_0000000 >> 7)
	f.PayloadLength = uint64(b1 & 0b0_1111111)
	if f.PayloadLength == 126 {
		payloadLen16, err := readNBytes(r, 2)
		if err != nil {
			return nil, fmt.Errorf("payload length 126 signaled that next 16 bits must be actual length, but failed to read them: [%w]", err)
		}
		f.PayloadLength = uint64(binary.BigEndian.Uint16(payloadLen16))
	} else if f.PayloadLength == 127 {
		payloadLen64, err := readNBytes(r, 8)
		if err != nil {
			return nil, fmt.Errorf("payload length 127 signaled that next 64 bits must be actual length, but failed to read them: [%w]", err)
		}
		f.PayloadLength = binary.BigEndian.Uint64(payloadLen64)
	}

	if f.Mask == MaskDataMasked {
		maskingKeyBytes, err := readNBytes(r, 4)
		if err != nil {
			return nil, fmt.Errorf("mask bit signaled that next 32 bits must have masking key, but failed to read them: [%w]", err)
		}
		f.MaskingKey = binary.BigEndian.Uint32(maskingKeyBytes)
	}

	// Extensions are not supported
	f.ExtensionData = nil

	f.ApplicationData, err = readNBytes(r, int(f.PayloadLength))
	if err != nil {
		return nil, fmt.Errorf("failed to %d bytes of frame data: [%w]", f.PayloadLength, err)
	}

	if f.Mask == MaskDataMasked {
		maskBytes(f.ApplicationData, f.MaskingKey)
	}

	return &f, nil
}

func (f *Frame) WriteFrame(w io.Writer) error {
	var b0, b1 byte

	b0 = b0 | byte(f.FIN)<<7
	b0 = b0 | byte(f.RSV1)<<6
	b0 = b0 | byte(f.RSV2)<<5
	b0 = b0 | byte(f.RSV3)<<4
	b0 = b0 | byte(f.Opcode)
	err := binary.Write(w, binary.BigEndian, b0)
	if err != nil {
		return fmt.Errorf("failed to write first byte of frame: [%w]", err)
	}

	b1 = b1 | byte(f.Mask)<<7

	if f.PayloadLength <= 125 {
		b1 = b1 | byte(f.PayloadLength)
		err = binary.Write(w, binary.BigEndian, b1)
	} else if f.PayloadLength >= math.MaxUint16 {
		b1 = b1 | byte(126)
		err = binary.Write(w, binary.BigEndian, b1)
		err = errors.Join(err, binary.Write(w, binary.BigEndian, uint16(f.PayloadLength)))
	} else {
		b1 = b1 | byte(127)
		err = binary.Write(w, binary.BigEndian, b1)
		err = errors.Join(err, binary.Write(w, binary.BigEndian, f.PayloadLength))
	}
	if err != nil {
		return fmt.Errorf("failed to write mask and payload length: [%w]", err)
	}

	if f.Mask == MaskDataMasked {
		err = binary.Write(w, binary.BigEndian, f.MaskingKey)
		if err != nil {
			return fmt.Errorf("failed to write mask key: [%w]", err)
		}
		maskBytes(f.ApplicationData, f.MaskingKey)
	}

	_, err = w.Write(f.ExtensionData)
	if err != nil {
		return fmt.Errorf("failed to write extension data: [%w]", err)
	}

	_, err = w.Write(f.ApplicationData)
	if err != nil {
		return fmt.Errorf("failed to write application data: [%w]", err)
	}

	return nil
}

// func (f *Frame) ToBytes() ([]byte, error) {
// 	// FIN + RSV1-3 + Opcode + MASK + PayloadLength (first 7 bits)
// 	var guaranteedFirst uint64 = 2

// 	payloadLengthExtensionSize := f.PayloadLength.Type.ExtensionBytesSize()

// 	maskingKeySize := f.Mask.MaskBytesSize()

// 	var totalLength uint64 = guaranteedFirst + uint64(payloadLengthExtensionSize) + uint64(maskingKeySize) + f.PayloadLength.Value

// 	bytes := make([]byte, totalLength)

// 	// var finRsvOpcodeByte byte

// 	return bytes, nil
// }

func maskBytes(bytes []byte, maskingKey uint32) {
	maskingKeyOctets := make([]byte, 0, 4)
	maskingKeyOctets = binary.BigEndian.AppendUint32(maskingKeyOctets, maskingKey)

	for i, b := range bytes {
		masked := b ^ maskingKeyOctets[i%4]
		bytes[i] = masked
	}
}
