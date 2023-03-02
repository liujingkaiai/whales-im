package iface

import (
	"github.com/klintcheng/kim/wire/pkt"
	"google.golang.org/protobuf/proto"
)

type ISession interface {
	GetChannelId() string
	GetGateId() string
	GetAccount() string
	GetZone() string
	GetIsp() string
	GetRemoteIP() string
	GetDevice() string
	GetApp() string
	GetTags() []string
}

type IContext interface {
	Dispatcher
	ISessionStorage
	Header() *pkt.Header
	ReadBody(proto.Message) error
	Session() ISession
	RespWithError(status pkt.Status, err error) error
	Resp(status pkt.Status, body proto.Message) error
	Dispatch(body proto.Message, recvs ...*Location) error
}
