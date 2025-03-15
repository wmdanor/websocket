package websocket

import (
	"github.com/wmdanor/websocket/go/internal"
)

type MessageType uint8

const (
	// Non-control
	TextMessage   MessageType = MessageType(internal.OpcodeTextFrame)
	BinaryMessage MessageType = MessageType(internal.OpcodeBinaryFrame)

	// Control
	CloseMessage MessageType = MessageType(internal.OpcodeConnectionClose)
	PingMessage  MessageType = MessageType(internal.OpcodePing)
	PongMessage  MessageType = MessageType(internal.OpcodePong)
)

// func (c *Conn) nextFrame() error {
// 	fixed := c.readNBytes(2)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to read first 2 essential bytes of the frame: [%w]", err)
// 	}
// 	b0, b1 := h[0], h[1]

// 	f := Frame{}
// 	f.FIN = FIN(b0 & 0b1_000_0000 >> 7)
// 	f.RSV1 = b0 & 0b0_100_0000 >> 6
// 	f.RSV2 = b0 & 0b0_010_0000 >> 5
// 	f.RSV3 = b0 & 0b0_001_0000 >> 4
// 	f.Opcode = Opcode(b0 & 0b0_000_1111)
// }
