package websocket

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"math/big"

	"github.com/wmdanor/websocket/go/internal"
)

func (c *Conn) WriteMessage(messageType MessageType, data []byte) error {
	if messageType != TextMessage && messageType != BinaryMessage {
		return fmt.Errorf("message type must be text or binary")
	}

	w, err := c.NextWriter(messageType)
	if err != nil {
		return err
	}

	_, err = w.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write data: [%w]", err)
	}

	return w.Close()
}

func (c *Conn) WriteClose(code CloseCode, message string) error {
	c.l.Debug("writing close message", "code", code, "data", message)
	return c.WriteControl(CloseMessage, CloseMessageData(code, message))
}

func (c *Conn) WriteControl(messageType MessageType, data []byte) error {
	if !internal.Opcode(messageType).IsControl() {
		return fmt.Errorf("message type must be close, ping or pong")
	}

	if messageType == CloseMessage && c.sentConnClose {
		c.l.Debug("Already wrote close message, skipping")
		return nil
	}
	if messageType == CloseMessage {
		c.sentConnClose = true
	}

	c.l.Debug("writing control frame", "messageType", messageType)
	err := c.writeFrame(true, internal.Opcode(messageType), data)
	if err != nil {
		return fmt.Errorf("failed to write control frame: [%w]", err)
	}

	return nil
}

func (c *Conn) NextWriter(messageType MessageType) (io.WriteCloser, error) {
	if c.err != nil {
		return nil, c.err
	}

	if c.sentConnClose || c.recvConnClose {
		return nil, fmt.Errorf("connection close was initiated")
	}

	if c.curWriter != nil {
		err := c.curWriter.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to close current writer: [%w]", err)
		}
	}

	c.wBuf.Reset()

	id, _ := rand.Int(rand.Reader, big.NewInt(1000))
	l := c.l.With("id", id.Int64())

	l.Debug("creating new writer", "messageType", messageType)

	c.curWriter = &messageWriter{
		c:           c,
		messageType: messageType,
		isFirst:     true,
		l:           l,
	}
	return c.curWriter, nil
}

type messageWriter struct {
	c *Conn

	messageType MessageType

	bytesReceived int

	isFirst bool
	isFinal bool

	l *slog.Logger
}

func (w *messageWriter) Write(p []byte) (n int, err error) {
	buf := w.c.wBuf
	written := 0

	l := w.l.With("buf.cap", buf.Cap(), "data.len", len(p))

	l.Debug("message writer: write")

	w.bytesReceived += len(p)
	if internal.Opcode(w.messageType).IsControl() && w.bytesReceived > 125 {
		return 0, fmt.Errorf("control messages must have application data less than 125, received %d", len(p)+w.bytesReceived)
	}

	for len(p) != 0 {
		if buf.Available() == 0 {
			l.Debug("message writer: write: buffer full, writing frame", "buf.len", buf.Len())
			err := w.writeFrame()
			if err != nil {
				return written, fmt.Errorf("failed to write frame: [%w]", err)
			}
		}

		l.Debug("message writer: write: writing to buffer", "buf.len", buf.Len(), "data.remaining", len(p))
		toCopy := min(len(p), buf.Available())
		n, _ := buf.Write(p[:toCopy])
		p = p[toCopy:]
		written += n
	}

	l.Debug("message writer: write: finished writing", "buf.len", buf.Len())
	return written, nil
}

func (w *messageWriter) Close() error {
	w.l.Debug("message writer: close", "buf.len", w.c.wBuf.Len())
	w.isFinal = true
	w.c.curWriter = nil

	return w.writeFrame()
}

func (w *messageWriter) writeFrame() error {
	w.l.Debug("writing frame")

	buf := w.c.wBuf

	isFirst := w.isFirst
	if w.isFirst {
		w.isFirst = false
	}

	defer buf.Reset()

	opcode := internal.OpcodeContinuationFrame
	if isFirst {
		opcode = internal.Opcode(w.messageType)
	}
	err := w.c.writeFrame(w.isFinal, opcode, buf.Bytes())

	return err
}

func (c *Conn) writeFrame(isFinal bool, opcode internal.Opcode, data []byte) error {
	if opcode.IsControl() && len(data) > 125 {
		return fmt.Errorf("control frame data must not exceed 125 bytes, received: %d", len(data))
	}

	dest := c.conn

	var b0, b1 byte

	if isFinal {
		b0 = b0 | 0b1_000_0000
	}
	// b0 = b0 | byte(0)<<6 // RSV1
	// b0 = b0 | byte(0)<<5 // RSV2
	// b0 = b0 | byte(0)<<4 // RSV3

	// Else continuation opcode is 0, so no need to write
	b0 = b0 | byte(opcode)

	c.l.Debug("frame byte 0", "binary", fmt.Sprintf("%08b", b0))

	err := binary.Write(dest, binary.BigEndian, b0)
	if err != nil {
		return c.fatal(CloseInternalServerErr,
			fmt.Errorf("failed to write first byte of frame: [%w]", err), "")
	}

	var maskingKey [4]byte
	if !c.isServer {
		b1 = b1 | 0b1_000_0000
		rand.Read(maskingKey[:])
	}

	if len(data) <= 125 {
		b1 = b1 | byte(len(data))
		err = binary.Write(dest, binary.BigEndian, b1)
	} else if len(data) <= math.MaxUint16 {
		b1 = b1 | byte(126)
		err = binary.Write(dest, binary.BigEndian, b1)
		err = errors.Join(err, binary.Write(dest, binary.BigEndian, uint16(len(data))))
	} else {
		b1 = b1 | byte(127)
		err = binary.Write(dest, binary.BigEndian, b1)
		err = errors.Join(err, binary.Write(dest, binary.BigEndian, uint64(len(data))))
	}
	c.l.Debug("frame byte 1", "binary", fmt.Sprintf("%08b", b1))
	if err != nil {
		return c.fatal(CloseInternalServerErr,
			fmt.Errorf("failed to write MASK and payload length: [%w]", err), "")
	}

	if !c.isServer {
		c.l.Debug("masking frame data", "maskingKey", maskingKey)
		_, err = dest.Write(maskingKey[:])
		if err != nil {
			return c.fatal(CloseInternalServerErr,
				fmt.Errorf("failed to write masking key: [%w]", err), "")
		}
		internal.Mask(data, maskingKey)
	}

	c.l.Debug("writing frame data", "data.len", len(data))
	_, err = dest.Write(data)
	if err != nil {
		return c.fatal(CloseInternalServerErr,
			fmt.Errorf("failed to write application data: [%w]", err), "")
	}
	c.l.Debug("wrote frame successfully")

	return nil
}
