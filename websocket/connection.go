package websocket

import (
	"im/iface"
	"net"

	"github.com/gobwas/ws"
)

type WsConn struct {
	net.Conn
}

func NewConn(conn net.Conn) *WsConn {
	return &WsConn{
		Conn: conn,
	}
}

func (c *WsConn) ReadFrame() (iface.IFrame, error) {
	f, err := ws.ReadFrame(c.Conn)
	if err != nil {
		return nil, err
	}
	return &Frame{raw: f}, nil
}

func (c *WsConn) WriteFrame(opcode iface.OpCode, data []byte) error {
	f := ws.NewFrame(ws.OpCode(opcode), true, data)
	return ws.WriteFrame(c.Conn, f)
}

func (c *WsConn) Flush() error {
	return nil
}
