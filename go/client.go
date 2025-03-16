package websocket

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
)

// TODO: add deadlines everywhere

type Dialer struct {
	Subprotocols []string

	InternalLogger *slog.Logger
}

func (d *Dialer) Dial(urlStr string, headers map[string]string) (*Conn, error) {
	l := d.InternalLogger
	if l == nil {
		l = slog.New(slog.DiscardHandler)
	}

	writeBuf := make([]byte, 4096)

	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to parse url: [%w]", ErrHandshakeFailure, err)
	}

	switch u.Scheme {
	case "ws":
		u.Scheme = "http"
	case "wss":
		u.Scheme = "https"
	default:
		return nil, fmt.Errorf("url schema must be ws or wss, actual %q", u.Scheme)
	}

	dialAddr := u.Host

	l.Debug("dialing websocket server")

	netConn, err := net.Dial("tcp", dialAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to dial remote address %q: [%w]", dialAddr, err)
	}
	defer func() {
		if netConn != nil {
			_ = netConn.Close()
		}
	}()

	req := http.Request{
		Method:     http.MethodGet,
		URL:        u,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Host:       u.Host,
		Header:     make(http.Header),
	}

	for hk, hv := range headers {
		req.Header[hk] = []string{hv}
	}

	req.Header[headerUpgrade] = []string{headerUpgradeExpected}
	req.Header[headerConn] = []string{headerConnExpected}
	req.Header[headerSecWsVersion] = []string{headerSecWsVersionExpected}

	if len(d.Subprotocols) > 0 {
		secWsProto := strings.Join(d.Subprotocols, ", ")
		req.Header[headerSecWsProto] = []string{secWsProto}
	}

	secWsKey := newSecWsKey()
	expectedSecWsAccept := newSecWebsocketAccept(secWsKey).String()

	req.Header[headerSecWsKey] = []string{secWsKey}

	err = req.Write(netConn)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to write request: [%w]", ErrHandshakeFailure, err)
	}

	bufReader := bufio.NewReaderSize(netConn, 4096)

	res, err := http.ReadResponse(bufReader, &req)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read response: [%w]", ErrHandshakeFailure, err)
	}
	res.Body = io.NopCloser(bytes.NewReader([]byte{}))

	if res.StatusCode != 101 {
		return nil, fmt.Errorf(`%w: status code must be %d , actual %d`,
			ErrHandshakeFailure, 101, res.StatusCode)
	}

	actual, ok := headerEquals(res.Header, headerUpgrade, headerUpgradeExpected)
	if !ok {
		return nil, fmt.Errorf(`%w: %q header must be %q , actual %q`,
			ErrHandshakeFailure, headerUpgrade, headerUpgradeExpected, actual)
	}

	actual, ok = headerEquals(res.Header, headerConn, headerConnExpected)
	if !ok {
		return nil, fmt.Errorf(`%w: %q header must be %q , actual %q`,
			ErrHandshakeFailure, headerConn, headerConnExpected, actual)
	}

	secWsAccept := res.Header.Get(headerSecWsAccept)
	if len(secWsAccept) == 0 {
		return nil, fmt.Errorf("%w: missing %q header", ErrHandshakeFailure, headerSecWsAccept)
	} else if secWsAccept != expectedSecWsAccept {
		return nil, fmt.Errorf("%w: %q header does not equal expected value", ErrHandshakeFailure, headerSecWsAccept)
	}

	c, err := newConn(netConn, bufReader, writeBuf, l)
	if err != nil {
		return nil, fmt.Errorf("failed to create conn object: [%w]", err)
	}

	netConn = nil

	return c, nil
}
