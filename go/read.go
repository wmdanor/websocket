package websocket

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"unicode/utf8"

	"github.com/wmdanor/websocket/go/internal"
)

func (c *Conn) NextMessage() (MessageType, []byte, error) {
	mt, data, err := c.NextReader()
	if err != nil {
		return MessageType(0), nil, fmt.Errorf("failed to get next reader: [%w]", err)
	}

	buf := bytes.Buffer{}
	_, err = buf.ReadFrom(data)
	if err != nil {
		return MessageType(0), nil, fmt.Errorf("failed to read data from reader: [%w]", err)
	}

	return mt, buf.Bytes(), nil
}

// If connection was closed, will return: errors.Is(err, io.EOF) == true
func (c *Conn) NextReader() (mt MessageType, data io.Reader, err error) {
	if c.err != nil {
		return MessageType(0), nil, c.err
	}

	if c.sentConnClose && c.recvConnClose {
		return MessageType(0), nil, fmt.Errorf("connection closed")
	}

	if c.curReader != nil {
		err := c.curReader.close()
		if err != nil {
			return MessageType(0), nil, fmt.Errorf("failed to close current reader: [%w]", err)
		}
		c.curReader = nil
	}

	f, err := c.readFrameHeader()
	if err != nil {
		return MessageType(0), nil, fmt.Errorf("failed to read frame: [%w]", err)
	}
	if f.Opcode == internal.OpcodeContinuationFrame {
		return MessageType(0), nil, c.fatal(CloseProtocolError,
			fmt.Errorf("first received frame must not be continuation frame: [%w]", err), "")
	}

	id, _ := rand.Int(rand.Reader, big.NewInt(1000))
	l := c.l.With("id", id.Int64())

	reader := messageReader{
		c:              c,
		messageType:    MessageType(f.Opcode),
		isFinal:        f.IsFinalFrame,
		bytesRemaining: int(f.PayloadLength),
		maskingKey:     f.MaskingKey,
		l:              l,
	}

	return MessageType(f.Opcode), &reader, nil
}

type messageReader struct {
	c *Conn

	messageType MessageType

	bytesRemaining int

	maskingKey [4]byte
	isFinal    bool

	l *slog.Logger
}

func (m *messageReader) Read(p []byte) (n int, err error) {
	if m.bytesRemaining == 0 && m.isFinal {
		return 0, io.EOF
	}

	maskOfset := 0

	for n < len(p) {
		m.l.Debug("reading data", "frame.bytesRemaining", m.bytesRemaining, "n", n, "p.len", len(p))
		if m.bytesRemaining == 0 {
			if m.isFinal {
				m.c.curReader = nil
				break
			}

			m.l.Debug("reading next frame")

			maskOfset = 0

			f, err := m.c.readFrameHeader()
			if err != nil {
				err = errors.Join(err, io.ErrUnexpectedEOF)
				return n, fmt.Errorf("failed to read frame: [%w]", err)
			}
			if f.Opcode != internal.OpcodeContinuationFrame {
				err = errors.Join(io.ErrUnexpectedEOF)
				return n, m.c.fatal(CloseProtocolError,
					fmt.Errorf("succeeding frames must be continuation frames received opcode: %X, [%w]", f.Opcode, err), "")
			}

			m.l.Debug("got next frame", "isFinal", f.IsFinalFrame, "maskingKey", f.MaskingKey, "payloadLength", f.PayloadLength)
			m.isFinal = f.IsFinalFrame
			m.maskingKey = f.MaskingKey
			m.bytesRemaining = int(f.PayloadLength)
		}

		nn, err := m.c.r.Read(p[n:min(len(p), n+m.bytesRemaining)])
		if err != nil {
			m.l.Debug("failed to read frame data chunk", "err", err)
			err = errors.Join(err, io.ErrUnexpectedEOF)
			return n, m.c.fatal(CloseInternalServerErr,
				fmt.Errorf("failed to read bytes: [%w]", err), "")
		}
		m.l.Debug("received frame data chunk", "bytes", nn)

		if m.c.isServer {
			m.l.Debug("unmasking data chunk", "maskingKey", m.maskingKey, "from", n, "to", n+nn)
			internal.MaskOffset(p[n:n+nn], m.maskingKey, maskOfset%4)
		}

		n += nn
		maskOfset += nn
		m.bytesRemaining -= nn
	}

	m.l.Debug("finished reading frame data chunk", "n", n)

	if m.messageType == TextMessage {
		// todo: what if read byte is not end and because of that utf8 validation fails for last rune
		valid := utf8.Valid(p[:n])
		if !valid {
			return n, m.c.fatal(CloseInvalidFramePayloadData,
				fmt.Errorf("received invalid UTF-8 data"), "")
		}
	}

	return n, nil
}

func (m *messageReader) close() error {
	_, err := io.Copy(io.Discard, m)
	if err != nil {
		return fmt.Errorf("failed to discard remaining current message: [%w]", err)
	}

	return nil
}

func (c *Conn) readFrameHeader() (*internal.FrameHeader, error) {
	for {
		c.l.Debug("reading frame header")

		f := internal.FrameHeader{}

		fixed, err := c.readNBytes(2)
		if err != nil {
			return nil, c.fatal(CloseInternalServerErr,
				fmt.Errorf("failed to read first 2 essential bytes of the frame: [%w]", err), "")
		}
		b0, b1 := fixed[0], fixed[1]

		f.IsFinalFrame = b0&0b1_000_0000 == 0b1_000_0000
		f.RSV1 = b0 & 0b0_100_0000 >> 6
		f.RSV2 = b0 & 0b0_010_0000 >> 5
		f.RSV3 = b0 & 0b0_001_0000 >> 4
		f.Opcode = internal.Opcode(b0 & 0b0_000_1111)

		if f.RSV1 != 0 || f.RSV2 != 0 || f.RSV3 != 0 {
			return nil, c.fatal(CloseProtocolError,
				fmt.Errorf("RSV bits must be 0 as extensions are not supported"), "")
		}

		c.l.Debug("read header byte 0", "partialHeader", f)

		if f.Opcode.IsReserved() {
			return nil, c.fatal(CloseProtocolError,
				fmt.Errorf("opcode must not be one of reserved values"), "")
		}

		f.IsMasked = b1&0b1_0000000 == 0b1_0000000
		if f.IsMasked && !c.isServer {
			return nil, c.fatal(CloseProtocolError,
				fmt.Errorf("received masked frame on the client"), "")
		}

		f.PayloadLength = uint64(b1 & 0b0_1111111)
		if f.PayloadLength == 126 {
			payloadLen16, err := c.readNBytes(2)
			if err != nil {
				return nil, c.fatal(CloseInternalServerErr,
					fmt.Errorf("payload length 126 signaled that next 16 bits must be actual length, but failed to read them: [%w]", err), "")
			}
			f.PayloadLength = uint64(binary.BigEndian.Uint16(payloadLen16))
		} else if f.PayloadLength == 127 {
			payloadLen64, err := c.readNBytes(8)
			if err != nil {
				return nil, c.fatal(CloseInternalServerErr,
					fmt.Errorf("payload length 127 signaled that next 64 bits must be actual length, but failed to read them: [%w]", err), "")
			}
			f.PayloadLength = binary.BigEndian.Uint64(payloadLen64)
		}

		c.l.Debug("read header byte 1 + payload len extra", "partialHeader", f)

		if f.Opcode.IsControl() && (f.PayloadLength > 125 || !f.IsFinalFrame) {
			return nil, c.fatal(CloseProtocolError,
				fmt.Errorf("all control frames must have a payload length of 125 bytes or less and must not be fragmented"), "")
		}

		if f.IsMasked {
			maskingKey, err := c.readNBytes(4)
			if err != nil {
				return nil, c.fatal(CloseInternalServerErr,
					fmt.Errorf("mask bit signaled that next 32 bits must have masking key, but failed to read them: [%w]", err), "")
			}
			copy(f.MaskingKey[:], maskingKey)
			c.l.Debug("read frame masking key", "partialHeader", f)
		}

		// Extensions are not supported
		f.ExtensionData = nil

		if f.Opcode.IsControl() {
			c.l.Debug("received frame is control, handling specially", "partialHeader", f)

			var buf []byte
			if f.PayloadLength != 0 {
				buf = make([]byte, f.PayloadLength)
				c.l.Debug("reading control frame data", "payloadLength", f.PayloadLength)
				_, err := io.ReadFull(c.r, buf) // TODO ???
				if err != nil {
					return nil, c.fatal(CloseInternalServerErr,
						fmt.Errorf("failed to read control frame data: [%w]", err), "")
				}
				if c.isServer {
					internal.Mask(buf, f.MaskingKey)
				}
			} else {
				c.l.Debug("control frame does not have data to read")
			}

			if f.Opcode == internal.OpcodeConnectionClose {
				c.l.Debug("received frame is close, handling specially")
				c.recvConnClose = true

				if f.PayloadLength == 1 {
					return nil, c.fatal(CloseProtocolError,
						fmt.Errorf("close frame must either have 0 or 2+ payload length, but received 1"), "")
				}
				if len(buf) > 2 && !utf8.Valid(buf[2:]) {
					return nil, c.fatal(CloseInvalidFramePayloadData,
						fmt.Errorf("close frame reason in data must be valid UTF-8 encoded string"), "")
				}

				closeCode := uint16(CloseNormalClosure)
				if len(buf) >= 2 {
					closeCode = binary.BigEndian.Uint16(buf)
					_, ok := NewCloseCode(closeCode)
					if !ok {
						return nil, c.fatal(CloseProtocolError,
							fmt.Errorf("received invalid close code: %d", closeCode), "")
					}
				}
				err = c.handleClose(CloseCode(closeCode), string(buf[min(len(buf), 2):]))
				if err != nil {
					return nil, fmt.Errorf("failed to handle close frame: [%w]", err)
				}
				c.err = fmt.Errorf("connection was closed: [%w]", io.EOF)
				return &f, c.err
			} else if f.Opcode == internal.OpcodePing {
				c.l.Debug("received frame is ping, handling specially")
				err = c.handlePing(buf)
				if err != nil {
					return nil, fmt.Errorf("failed to handle ping frame: [%w]", err)
				}
			} else if f.Opcode == internal.OpcodePong {
				c.l.Debug("received frame is pong, handling specially")
				err = c.handlePong(buf)
				if err != nil {
					return nil, fmt.Errorf("failed to handle pong frame: [%w]", err)
				}
			}

			continue
		}

		return &f, nil
	}
}

// TODO; this is shit : The bytes stop being valid at the next read call.
func (c *Conn) readNBytes(n int) ([]byte, error) {
	bytes, err := c.r.Peek(n)
	if err == io.EOF {
		return nil, io.ErrUnexpectedEOF
	}

	c.r.Discard(len(bytes))

	return bytes, err
}
