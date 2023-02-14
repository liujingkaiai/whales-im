package iface

import (
	"context"
	"time"
)

type OpCode int

const (
	DefaultLoginWait time.Duration = time.Second * 3
	DefaultReadWait  time.Duration = time.Second * 3
)

// server.go
const (
	OpContinuation OpCode = 0x0
	OpText         OpCode = 0x1
	OpBinary       OpCode = 0x2
	OpClose        OpCode = 0x8
	OpPing         OpCode = 0x9
	OpPong         OpCode = 0xa
)

type IServer interface {
	ServiceRegistration
	//设置握手处理
	SetAcceptor(IAcceptor)
	//设置消息监听器
	SetMessageListener(IMessageListener)
	//设置用户监听器
	SetStateListener(IStatelistener)
	//设置读超时
	SetReadWait(time.Duration)
	//设置连接管理器
	SetChannelMap(IChannelMap)

	Start() error
	Push(string, []byte) error
	Shutdown(context.Context) error
}

//服务接受者
type IAcceptor interface {
	//返回一个id
	Accept(IConn, time.Duration) (string, error)
}

//断开连接回调函数
type IStatelistener interface {
	Disconnect(string) error
}

type IService interface {
	ServiceID() string
	ServiceName() string
	GetMeta() map[string]string
}
