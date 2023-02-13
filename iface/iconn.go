package iface

import "net"

//连接
type IConn interface {
	net.Conn
	//读取消息帧
	ReadFrame() (IFrame, error)
	WriteFrame(OpCode, []byte) error
	Flush() error
}
