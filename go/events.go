package websocket

import (
	"fmt"
)

func (c *Conn) SetCloseHandler(h func(code CloseCode, appData string) error) {
	if h == nil {
		c.handleClose = func(code CloseCode, appData string) error {
			c.l.Debug("received close message", "code", code, "data", appData)
			err := c.WriteClose(code, appData)
			if err != nil {
				return fmt.Errorf("failed to write close message: [%w]", err)
			}

			return nil
		}
	} else {
		c.handleClose = h
	}
}

func (c *Conn) SetPingHandler(h func(appData []byte) error) {
	if h == nil {
		c.handlePing = func(appData []byte) error {
			c.l.Debug("received ping message", "strdata", string(appData))
			err := c.WriteControl(PongMessage, appData)
			if err != nil {
				return fmt.Errorf("failed to write pong message: [%w]", err)
			}

			return nil
		}
	} else {
		c.handlePing = h
	}
}

func (c *Conn) SetPongHandler(h func(appData []byte) error) {
	if h == nil {
		c.handlePong = func(appData []byte) error {
			c.l.Debug("received pong message", "strdata", string(appData))
			return nil
		}
	} else {
		c.handlePong = h
	}
}
