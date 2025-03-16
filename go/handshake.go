package websocket

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
)

const (
	headerHost         = "Host"
	headerUpgrade      = "Upgrade"
	headerConn         = "Connection"
	headerSecWsVersion = "Sec-WebSocket-Version"
	headerSecWsProto   = "Sec-WebSocket-Protocol"
	headerSecWsExt     = "Sec-WebSocket-Extensions"
	headerSecWsKey     = "Sec-WebSocket-Key"
	headerSecWsAccept  = "Sec-WebSocket-Accept"

	headerUpgradeExpected      = "websocket"
	headerConnExpected         = "Upgrade"
	headerSecWsVersionExpected = "13"

	wsGuid = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
)

var (
	ErrHandshakeFailure = errors.New("handshake failure")
)

func newSecWsKey() string {
	nonce := [16]byte{}

	rand.Read(nonce[:])

	b64 := base64.StdEncoding.EncodeToString(nonce[:])

	return b64
}

type secWebsocketAccept string

func newSecWebsocketAccept(secWebSocketKey string) secWebsocketAccept {
	concat := secWebSocketKey + wsGuid

	hasher := sha1.New()
	hasher.Write([]byte(concat))

	bytes := hasher.Sum(nil)
	b64 := base64.StdEncoding.EncodeToString(bytes)

	return secWebsocketAccept(b64)
}

func (a secWebsocketAccept) String() string {
	return string(a)
}

// Checks if header equals expected value (case insensitive)
// If yes - returns `"", true`
// If no - returns `"<actual_value>", false`
func headerEquals(h http.Header, header, expectedValue string) (string, bool) {
	actualValue := h.Get(header)
	if strings.EqualFold(expectedValue, actualValue) {
		return "", true
	} else {
		return actualValue, false
	}
}
