package iface

import (
	"net"
	"time"
)

const (
	DefaultWriteWait time.Duration = 3 * time.Second
)

//客户端
type IClient interface {
	ID() string
	Name() string
	//连接服务端
	Connect(string) error
	//设置拨号器
	SetDialer(IDialer)
	Send([]byte) error
	Read() (IFrame, error)
	Close()
}

type IDialer interface {
	DialAndHandshake(DialerContext) (net.Conn, error)
}

type DialerContext struct {
	Id      string
	Name    string
	Address string
	Timeout time.Duration
}
