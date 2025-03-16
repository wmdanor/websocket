package websocket

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"
)

type Upgrader struct {
	InternalLogger *slog.Logger
}

// On a server call this in your http handler
// to upgrade connection to Websocket connection
func (u *Upgrader) Upgrade(w http.ResponseWriter, req *http.Request) (*Conn, error) {
	l := u.InternalLogger
	if l == nil {
		l = slog.New(slog.DiscardHandler)
	}

	l.Debug("Opening new websocket connection")

	err := u.handleOpenHandshake(w, req, l)
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

	conn, err := newConn(netConn, rw.Reader, rw.AvailableBuffer(), l)
	if err != nil {
		return nil, err
	}

	conn.isServer = true

	return conn, nil
}

func (u *Upgrader) handleOpenHandshake(w http.ResponseWriter, req *http.Request, l *slog.Logger) error {
	w.Header().Add("Access-Control-Allow-Origin", "*")

	l.Debug("Handling opening handshake")

	if req.Method != http.MethodGet {
		return fmt.Errorf("%w: method must be %q, actual %q",
			ErrHandshakeFailure, http.MethodGet, req.Method)
	}

	// Check Host header

	actual, ok := headerEquals(req.Header, headerUpgrade, headerUpgradeExpected)
	if !ok {
		return fmt.Errorf(`%w: %q header must be %q , actual %q`,
			ErrHandshakeFailure, headerUpgrade, headerUpgradeExpected, actual)
	}

	actual, ok = headerEquals(req.Header, headerConn, headerConnExpected)
	if !ok {
		return fmt.Errorf(`%w, %q header must be %q, actual: %q`,
			ErrHandshakeFailure, headerConn, headerConnExpected, actual)
	}

	actual, ok = headerEquals(req.Header, headerSecWsVersion, headerSecWsVersionExpected)
	if !ok {
		return fmt.Errorf(`%w, %q header must be %q, received: %q`,
			ErrHandshakeFailure, headerSecWsVersion, headerSecWsVersion, actual)
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
		return fmt.Errorf("%w: missing %q header", ErrHandshakeFailure, headerSecWsKey)
	} else {
		decoded, err := base64.StdEncoding.DecodeString(secWsKey)
		if err != nil {
			return fmt.Errorf("%w: failed to base64 decode %q header: [%w]",
				ErrHandshakeFailure, headerSecWsKey, err)
		}
		if len(decoded) != 16 {
			return fmt.Errorf("%w: decoded value of %q must be 16 bytes, received %d bytes",
				ErrHandshakeFailure, headerSecWsKey, len(decoded))
		}
	}

	secWebsocketAccept := newSecWebsocketAccept(secWsKey)

	w.Header().Add(headerUpgrade, headerUpgradeExpected)
	w.Header().Add(headerConn, headerConnExpected)
	w.Header().Add(headerSecWsAccept, secWebsocketAccept.String())

	w.WriteHeader(101)

	return nil
}
