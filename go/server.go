package websocket

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"
	"os"
)

// On a server call this in your http handler
// to upgrade connection to Websocket connection
func UpgradeConnection(w http.ResponseWriter, req *http.Request) (*Conn, error) {
	l := slog.New(slog.DiscardHandler)
	if os.Getenv("WS_LOG") == "1" {
		l = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
	}

	l.Debug("Opening new websocket connection")

	err := handleOpenHandshake(w, req, l)
	if err != nil {
		l.Debug(fmt.Sprintf("Failed to open websocket connection: %s", err.Error()))
		return nil, err
	}

	netConn, rw, err := http.NewResponseController(w).Hijack()
	if err != nil {
		l.Debug("Failed to open websocket connection: couldn't hijack TCP connection")
		return nil, fmt.Errorf("failed to hijack net.Conn")
	}

	l.Debug("New websocket connection opened")

	conn, err := newConn(netConn, rw.Reader, rw.AvailableBuffer())
	if err != nil {
		return nil, err
	}

	conn.isServer = true

	return conn, nil
}

func handleOpenHandshake(w http.ResponseWriter, req *http.Request, l *slog.Logger) error {
	w.Header().Add("Access-Control-Allow-Origin", "*")

	l.Debug("Handling opening handshake")

	if req.Method != http.MethodGet {
		return fmt.Errorf("%w: method must be GET, actual %q",
			ErrInvalidHandshakeRequest, req.Method)
	}

	// Check Host header

	actual, ok := headerEquals(req, headerUpgrade, headerUpgradeExpected)
	if !ok {
		return fmt.Errorf(`%w: %q header must be %q , actual %q`,
			ErrInvalidHandshakeRequest, headerUpgrade, headerUpgradeExpected, actual)
	}

	actual, ok = headerEquals(req, headerConn, headerConnExpected)
	if !ok {
		return fmt.Errorf(`%w, %q header must be %q, actual: %q`,
			ErrInvalidHandshakeRequest, headerConn, headerConnExpected, actual)
	}

	actual, ok = headerEquals(req, headerSecWsVersion, headerSecWsVersionExpected)
	if !ok {
		return fmt.Errorf(`%w, %q header must be %q, received: %q`,
			ErrInvalidHandshakeRequest, headerSecWsVersion, headerSecWsVersion, actual)
	}

	secWsProto := req.Header.Get(headerSecWsProto)
	if secWsProto != "" {
		// TODO
		l.Debug(fmt.Sprintf(`%q header not supported, received value: %q`, headerSecWsProto, secWsProto))
	}

	secWsExt := req.Header.Get(headerSecWsExt)
	if secWsExt != "" {
		// TODO
		l.Debug(fmt.Sprintf(`%q header not supported, received value: %q`, headerSecWsExt, secWsExt))
	}

	secWsKey := req.Header.Get(headerSecWsKey)
	if len(secWsKey) == 0 {
		return fmt.Errorf("%w: missing %q header", ErrInvalidHandshakeRequest, headerSecWsKey)
	} else {
		decoded, err := base64.StdEncoding.DecodeString(secWsKey)
		if err != nil {
			return fmt.Errorf("%w: failed to base64 decode %q header: [%w]",
				ErrInvalidHandshakeRequest, headerSecWsKey, err)
		}
		if len(decoded) != 16 {
			return fmt.Errorf("%w: decoded value of %q must be 16 bytes, received %d bytes",
				ErrInvalidHandshakeRequest, headerSecWsKey, len(decoded))
		}
	}

	secWebsocketAccept := newSecWebsocketAccept(secWsKey)

	w.Header().Add(headerUpgrade, headerUpgradeExpected)
	w.Header().Add(headerConn, headerConnExpected)
	w.Header().Add(headerSecWsAccept, secWebsocketAccept.String())

	w.WriteHeader(101)

	return nil
}
