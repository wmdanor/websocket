package websocket

import (
	"errors"
	"fmt"
	"slices"
	"time"
)

type CloseCode uint16

// Close codes defined in RFC 6455, section 11.7.
const (
	CloseNormalClosure           CloseCode = 1000
	CloseGoingAway               CloseCode = 1001
	CloseProtocolError           CloseCode = 1002
	CloseUnsupportedData         CloseCode = 1003
	CloseNoStatusReceived        CloseCode = 1005
	CloseAbnormalClosure         CloseCode = 1006
	CloseInvalidFramePayloadData CloseCode = 1007
	ClosePolicyViolation         CloseCode = 1008
	CloseMessageTooBig           CloseCode = 1009
	CloseMandatoryExtension      CloseCode = 1010
	CloseInternalServerErr       CloseCode = 1011
	CloseServiceRestart          CloseCode = 1012
	CloseTryAgainLater           CloseCode = 1013
	CloseTLSHandshake            CloseCode = 1015
)

var (
	validCloseCodes []CloseCode = []CloseCode{
		CloseNormalClosure,
		CloseGoingAway,
		CloseProtocolError,
		CloseUnsupportedData,
		CloseInvalidFramePayloadData,
		ClosePolicyViolation,
		CloseMessageTooBig,
		CloseMandatoryExtension,
		CloseInternalServerErr,
		CloseServiceRestart,
		CloseTryAgainLater,
		// CloseTLSHandshake, // ???
	}
)

func (c CloseCode) U() uint16 {
	return uint16(c)
}

func NewCloseCode(code uint16) (c CloseCode, ok bool) {
	c = CloseCode(code)
	ok = c.IsValid()
	return
}

func (c CloseCode) IsValid() bool {
	defined := slices.Contains(validCloseCodes, c)

	range3k4k := false
	if c >= 3000 && c <= 4999 {
		range3k4k = true
	}

	return defined || range3k4k
}

func (c *Conn) Close() error {
	return c.close(CloseNormalClosure, "")
}

// you are not supposed to check returned err as it will just join close err and passed err
// use to close conn and store last err
func (c *Conn) fatal(code CloseCode, err error, message string) error {
	c.l.Debug("connection fatal error, closing connection", "err", err)
	c.err = err

	if message == "" {
		message = err.Error()
	}

	return errors.Join(err, c.close(code, message))
}

func (c *Conn) close(code CloseCode, message string) error {
	defer c.conn.Close()

	if c.sentConnClose && c.recvConnClose {
		c.l.Debug("Already sent and received close frames, connection is closed, skipping")
		return nil
	}

	c.l.Debug("Closing websocket connection")

	err := c.WriteClose(code, message)
	if err != nil {
		c.l.Debug("Failed to send close frame", "err", err)
		return err
	}

	err = c.waitCloseFrame()
	if err != nil {
		c.l.Debug("Failed to receive close frame", "err", err)
		return err
	}

	c.l.Debug("Websocket connection closed")

	return nil
}

func (c *Conn) waitCloseFrame() error {
	if c.recvConnClose {
		c.l.Debug("Close frame already received, skipping")
		return nil
	}

	timeout := 15 * time.Second
	err := c.conn.SetReadDeadline(time.Now().Add((timeout)))
	if err != nil {
		return fmt.Errorf("failed to set timeout for socker read: [%w]", err)
	}

	deadline := time.Now().Add(time.Second * 15)
	for {
		if time.Now().Compare(deadline) == 1 {
			return fmt.Errorf("reached wait for close frame deadline")
		}

		c.l.Debug("Waiting for close frame")
		mt, _, err := c.NextReader()
		if err != nil {
			return fmt.Errorf("failed to read frame while waiting for close frame: [%w]", err)
		}
		if mt == CloseMessage {
			c.l.Debug("Received close frame")
			return nil
		}
		c.l.Debug("Received non-close frame")
	}
}
