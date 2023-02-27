package iface

import "time"

type IChannel interface {
	IConn
	IAgent
	//关闭
	Close() error
	//连续读取消息
	Readloop(lst IMessageListener) error
	// SetWriteWait 设置写超时
	SetWriteWait(time.Duration)
	SetReadWait(time.Duration)
}
