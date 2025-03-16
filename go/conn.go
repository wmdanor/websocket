package websocket

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"log/slog"
	"net"
)

type Conn struct {
	l *slog.Logger

	conn net.Conn

	r    *bufio.Reader
	wBuf *bytes.Buffer

	sentConnClose bool
	recvConnClose bool

	curReader *messageReader
	curWriter *messageWriter

	isServer bool

	handleClose func(code CloseCode, appData string) error
	handlePing  func(appData []byte) error
	handlePong  func(appData []byte) error

	err error
}

const (
	// min size to be able to store control messages data
	minWriteBufSize = 4096
)

func newConn(netConn net.Conn, reader *bufio.Reader, writeBuf []byte, l *slog.Logger) (*Conn, error) {
	if len(writeBuf) < minWriteBufSize {
		writeBuf = make([]byte, minWriteBufSize)
	}

	conn := &Conn{
		conn: netConn,
		r:    reader,
		wBuf: bytes.NewBuffer(writeBuf),
		l:    l,
	}

	conn.SetCloseHandler(nil)
	conn.SetPingHandler(nil)
	conn.SetPongHandler(nil)

	return conn, nil
}

func CloseMessageData(code CloseCode, message string) []byte {
	msgB := []byte(message)
	b := make([]byte, 2+len(msgB))

	binary.BigEndian.PutUint16(b, code.U())

	return b
}
