package tcp

import (
	"im/iface"
	"im/wire/endian"
	"io"
	"net"
)

type TcpConn struct {
	net.Conn
}

func NewTcpConn(conn net.Conn) *TcpConn {
	return &TcpConn{
		Conn: conn,
	}
}

func (c *TcpConn) ReadFrame() (iface.IFrame, error) {
	opcode, err := endian.ReadUint8(c.Conn)
	if err != nil {
		return nil, err
	}
	payload, err := endian.ReadBytes(c.Conn)
	if err != nil {
		return nil, err
	}
	return &Frame{
		OpCode:  iface.OpCode(opcode),
		Payload: payload,
	}, nil
}

func (c *TcpConn) WriteFrame(code iface.OpCode, payload []byte) error {
	return WriteFrame(c.Conn, code, payload)
}

func (c *TcpConn) Flush() error {
	return nil
}

func WriteFrame(w io.Writer, code iface.OpCode, payload []byte) error {
	if err := endian.WriteUint8(w, uint8(code)); err != nil {
		return err
	}
	if err := endian.WriteBytes(w, payload); err != nil {
		return err
	}
	return nil
}
