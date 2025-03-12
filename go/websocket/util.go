package websocket

import (
	"crypto/sha1"
	"encoding/base64"
	"net/http"
	"strings"
)

type secWebsocketAccept string

func newSecWebsocketAccept(secWebSocketKey string) secWebsocketAccept {
	guid := "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	concat := secWebSocketKey + guid

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
func headerEquals(req *http.Request, header, expectedValue string) (string, bool) {
	actualValue := req.Header.Get(header)
	if strings.EqualFold(expectedValue, actualValue) {
		return "", true
	} else {
		return actualValue, false
	}
}
