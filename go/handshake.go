package websocket

const (
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
)
