package frame

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"reflect"
	"testing"
)

type frameTestCase struct {
	bytes []byte
	frame Frame
}

var (
	frameTestCases []frameTestCase = []frameTestCase{
		{
			bytes: []byte{
				0x81, 0x85, 0x37, 0xFA, 0x21, 0x3D, 0x7F, 0x9F, 0x4D, 0x51, 0x58,
			},
			frame: Frame{
				FIN:             FINFinalFrame,
				RSV1:            0,
				RSV2:            0,
				RSV3:            0,
				Opcode:          OpcodeTextFrame,
				Mask:            MaskDataMasked,
				PayloadLength:   5,
				MaskingKey:      0x37FA213D,
				ExtensionData:   nil,
				ApplicationData: []byte("Hello"),
			},
		},
		{
			bytes: []byte{
				0x81, 0x05, 0x57, 0x6F, 0x72, 0x6C, 0x64,
			},
			frame: Frame{
				FIN:             FINFinalFrame,
				RSV1:            0,
				RSV2:            0,
				RSV3:            0,
				Opcode:          OpcodeTextFrame,
				Mask:            MaskDataNotMasked,
				PayloadLength:   5,
				MaskingKey:      0,
				ExtensionData:   nil,
				ApplicationData: []byte("World"),
			},
		},
	}
)

func TestFrameReadWrite(t *testing.T) {
	for _, individualTest := range frameTestCases {
		bytesCopy := make([]byte, len(individualTest.bytes))
		copy(bytesCopy, individualTest.bytes)

		testFrameReadWrite(
			t,
			bufio.NewReader(bytes.NewReader(bytesCopy)),
			individualTest.frame,
			individualTest.bytes,
		)
	}
}

func TestFrameReadWriteSequential(t *testing.T) {
	testFrameReadWriteSequential(t, frameTestCases)
}

func testFrameReadWriteSequential(t *testing.T, testCases []frameTestCase) {
	var sequence []byte
	for _, testCase := range testCases {
		clone := make([]byte, len(testCase.bytes))
		copy(clone, testCase.bytes)
		sequence = append(sequence, clone...)
	}
	reader := bufio.NewReader(bytes.NewReader(sequence))

	for _, testCase := range testCases {
		testFrameReadWrite(t, reader, testCase.frame, testCase.bytes)
	}
}

func testFrameReadWrite(t *testing.T, r *bufio.Reader, frame Frame, data []byte) {
	encoded, err := ReadFrame(r)
	if err != nil {
		t.Errorf("ReadFrame(%v), ERROR returned unexpected error %q", data, err.Error())
	}

	if !reflect.DeepEqual(*encoded, frame) {
		t.Errorf("ReadFrame(%v) = %+v, ERROR expected %+v", data, encoded, frame)
	} else {
		t.Logf("ReadFrame(%v) = %+v, OK", data, encoded)
	}

	var buf bytes.Buffer

	err = encoded.WriteFrame(&buf)
	decoded, bErr := io.ReadAll(&buf)
	err = errors.Join(err, bErr)
	if err != nil {
		t.Errorf("Frame.WriteFrame(%+v), ERROR returned unexpected error %q", encoded, err.Error())
	}

	if !reflect.DeepEqual(decoded, data) {
		t.Errorf("Frame.WriteFrame(%+v) => %v, ERROR expected %v", encoded, decoded, data)
	} else {
		t.Logf("Frame.WriteFrame(%+v) => %v, OK", encoded, decoded)
	}
}

func TestMask(t *testing.T) {
	var maskingKey uint32 = 0x12345678
	initial := []byte("Hello")
	toProcess := make([]byte, len(initial))
	copy(toProcess, initial)

	maskBytes(toProcess, maskingKey)

	expected := []byte{
		0x5A, 0x51, 0x3A, 0x14, 0x7D,
	}
	if !reflect.DeepEqual(toProcess, expected) {
		t.Errorf("maskBytes(%v, %X) => %v, ERROR expected %v", initial, maskingKey, toProcess, expected)
	} else {
		t.Logf("maskBytes(%v, %X) => %v, OK", initial, maskingKey, toProcess)
	}

	maskBytes(toProcess, maskingKey)

	if !reflect.DeepEqual(toProcess, initial) {
		t.Errorf("maskBytes(maskBytes(%v, %X), %X) => %v, ERROR expected %v", initial, maskingKey, maskingKey, toProcess, expected)
	} else {
		t.Logf("maskBytes(maskBytes(%v, %X), %X) => %v, OK", initial, maskingKey, maskingKey, toProcess)
	}
}

func TestBitWrite(t *testing.T) {
	var b0, b1 byte

	t.Logf("b0 %08b", b0)

	f := Frame{
		FIN:           FIN(1),
		Opcode:        OpcodeTextFrame,
		Mask:          1,
		PayloadLength: 34,
	}

	b0 = b0 | byte(f.FIN)<<7
	b0 = b0 | byte(f.RSV1)<<6
	b0 = b0 | byte(f.RSV2)<<5
	b0 = b0 | byte(f.RSV3)<<4

	b0 = b0 | byte(f.Opcode)

	t.Logf("b0 %08b", b0)

	t.Logf("b1 %08b", b1)

	b1 = b1 | byte(f.Mask)<<7
	b1 = b1 | byte(f.PayloadLength)

	t.Logf("b1 %08b", b1)
}
