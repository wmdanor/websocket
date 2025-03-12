package websocket

import (
	"bufio"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"

	"github.com/wmdanor/websocket/go/websocket/frame"

	"go.uber.org/zap"
)

var (
	ErrInvalidHandshakeRequest = errors.New("invalid handshake request")
)

type Conn struct {
	conn net.Conn
	rw   *bufio.ReadWriter

	sentConnClose bool
	recvConnClose bool

	l *slog.Logger
}

func must[T any](obj T, err error) T {
	if err != nil {
		panic(err)
	}
	return obj
}

func NewConnection(w http.ResponseWriter, req *http.Request) (*Conn, error) {
	l := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	err := handleOpenHandshake(w, req)
	if err != nil {
		return nil, err
	}

	netConn, rw, err := http.NewResponseController(w).Hijack()
	if err != nil {
		return nil, fmt.Errorf("failed to hijack net.Conn")
	}

	return &Conn{
		conn: netConn,
		rw:   rw,
		l:    must(zap.NewDevelopment()),
	}, nil
}

func (c *Conn) ReadFrame() (*frame.Frame, error) {
	f, err := frame.ReadFrame(c.rw.Reader)
	if err != nil {
		return nil, err
	}

	if f.Opcode == frame.OpcodeConnectionClose {
		c.recvConnClose = true
		err := c.Close()
		return nil, fmt.Errorf("connection is closed because recevied close frame: [%w]", err)
	}

	return f, nil
}

func (c *Conn) WriteFrame(f *frame.Frame) error {
	// if c.isClosingClosed() {
	// 	return errors.New("connection is closing or closed")
	// }

	return f.WriteFrame(c.rw.Writer)
}

func (c *Conn) Close() error {
	defer c.rw.Flush()
	defer c.conn.Close()

	if !c.sentConnClose {
		err := c.sendCloseFrame()
		if err != nil {
			return err
		}
	}

	if !c.recvConnClose {
		err := c.waitCloseFrame()
		if err != nil {
			return err
		}
	}

	fmt.Println("conn closed")

	return nil
}

func (c *Conn) sendCloseFrame() error {
	if c.sentConnClose {
		return nil
	}

	closeFrame := frame.Frame{
		FIN:    frame.FINFinalFrame,
		Opcode: frame.OpcodeConnectionClose,
	}
	fmt.Println("sending close frame")
	err := closeFrame.WriteFrame(c.rw.Writer)
	c.sentConnClose = true
	if err != nil {
		return fmt.Errorf("failed to send connection close frame: [%w]", err)
	}

	return nil
}

func (c *Conn) waitCloseFrame() error {
	if c.recvConnClose {
		return nil
	}

	for {
		fmt.Println("waiting for close frame")
		f, err := c.ReadFrame()
		fmt.Println("7878877878")
		if err != nil {
			return fmt.Errorf("failed to read frame while waiting for close frame: [%w]", err)
		}
		if f.Opcode == frame.OpcodeConnectionClose {
			return nil
		}
	}
}

func (c *Conn) isClosingClosed() bool {
	return c.sentConnClose || c.recvConnClose
}

func handleOpenHandshake(w http.ResponseWriter, req *http.Request, l *slog.Logger) error {
	w.Header().Add("Access-Control-Allow-Origin", "*")

	l.Debug("Handling opening handshake")

	if req.Method != http.MethodGet {
		l.Debug("Opening handshake sent request with non GET method, rejecting")
		return fmt.Errorf("%w: method must be GET", ErrInvalidHandshakeRequest)
	}

	// Check Host header

	actual, ok := headerEquals(req, "Upgrade", "websocket")
	if !ok {
		l.Debug("")
		return fmt.Errorf("%w, Upgrade header must be websocket , received: %s", ErrInvalidHandshakeRequest, actual)
	}

	actual, ok = headerEquals(req, "Connection", "Upgrade")
	if !ok {
		return fmt.Errorf("%w, Connection header must be Upgrade, received: %s", ErrInvalidHandshakeRequest, actual)
	}

	actual, ok = headerEquals(req, "Sec-WebSocket-Version", "13")
	if !ok {
		return fmt.Errorf("%w, Sec-WebSocket-Version header must be 13, received: %s", ErrInvalidHandshakeRequest, actual)
	}

	secWsProtocol := req.Header.Get("Sec-WebSocket-Protocol")
	if secWsProtocol != "" {
		// TODO
		fmt.Printf("DEBUG Sec-WebSocket-Protocol not supported yet, received: %s\n", secWsProtocol)
	}

	secWsExtensions := req.Header.Get("Sec-WebSocket-Extensions")
	if secWsExtensions != "" {
		// TODO
		fmt.Printf("DEBUG Sec-WebSocket-Extensions not supported yet, received: %s\n", secWsExtensions)
	}

	secKeyHeader := req.Header.Get("Sec-WebSocket-Key")
	if len(secKeyHeader) == 0 {
		return fmt.Errorf("%w: missing Sec-WebSocket-Key header", ErrInvalidHandshakeRequest)
	} else {
		decoded, err := base64.StdEncoding.DecodeString(secKeyHeader)
		if err != nil {
			return fmt.Errorf("%w: failed to base64 decode Sec-WebSocket-Key header", ErrInvalidHandshakeRequest)
		}
		if len(decoded) != 16 {
			return fmt.Errorf("%w: decoded value of Sec-WebSocket-Key must be 16 bytes, received %d bytes",
				ErrInvalidHandshakeRequest, len(decoded))
		}
	}

	secWebsocketAccept := newSecWebsocketAccept(secKeyHeader)

	w.Header().Add("Upgrade", "websocket")
	w.Header().Add("Connection", "Upgrade")
	w.Header().Add("Sec-WebSocket-Accept", secWebsocketAccept.String())

	w.WriteHeader(101)

	return nil
}
