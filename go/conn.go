package websocket

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"log/slog"
	"net"
	"os"
)

var (
	ErrInvalidHandshakeRequest = errors.New("invalid handshake request")
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

func newConn(netConn net.Conn, reader *bufio.Reader, writeBuf []byte) (*Conn, error) {
	l := slog.New(slog.DiscardHandler)
	if os.Getenv("WS_LOG") == "1" {
		w := os.Stdout
		if os.Getenv("WS_LOG_FILE") != "" {
			f, err := os.Create(os.Getenv("WS_LOG_FILE"))
			if err != nil {
				return nil, err
			}
			w = f
		}
		l = slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
	}

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
