package core

import (
	"im/iface"
	"im/logger"
	"sync"

	"github.com/klintcheng/kim/wire"
	"github.com/klintcheng/kim/wire/pkt"
	"google.golang.org/protobuf/proto"
)

type HandleFunc func(iface.IContext)

type HandleFuncChain []HandleFunc

type Context struct {
	sync.Mutex
	iface.Dispatcher
	iface.ISessionStorage

	handlers HandleFuncChain
	index    int
	request  *pkt.LogicPkt
	session  iface.ISession
}

func BuildContext() iface.IContext {
	return &Context{}
}

func (c *Context) Next() {
	if c.index >= len(c.handlers) {
		return
	}
	f := c.handlers[c.index]
	c.index++
	if f == nil {
		logger.Warn("arrived unknown HandlerFunc")
		return
	}
	f(c)
}

func (c *Context) RespWithError(status pkt.Status, err error) error {
	return c.Resp(status, &pkt.ErrorResp{Message: err.Error()})
}

func (c *Context) Resp(status pkt.Status, body proto.Message) error {
	packet := pkt.NewFrom(&c.request.Header)
	packet.Status = status
	packet.WriteBody(body)
	packet.Flag = pkt.Flag_Response

	logger.Debugf("<-- Resp to %s command:%s  status: %v body: %s", c.Session().GetAccount(), &c.request.Header, status, body)
	err := c.Push(c.Session().GetGateId(), []string{c.Session().GetChannelId()}, packet)
	if err != nil {
		logger.Error(err)
	}
	return err
}

func (c *Context) Dispatch(body proto.Message, revcs ...*iface.Location) error {
	if len(revcs) == 0 {
		return nil
	}
	packet := pkt.NewFrom(&c.request.Header)
	packet.Flag = pkt.Flag_Push
	packet.WriteBody(body)

	logger.Debugf("<-- Dispatch to %d users command:%s", len(revcs), &c.request.Header)
	group := make(map[string][]string)
	for _, revc := range revcs {
		if revc.ChannelID == c.Session().GetChannelId() {
			continue
		}
		if _, ok := group[revc.GateId]; !ok {
			group[revc.GateId] = make([]string, 0)
		}
		group[revc.GateId] = append(group[revc.GateId], revc.ChannelID)
	}

	for gateway, ids := range group {
		err := c.Push(gateway, ids, packet)
		if err != nil {
			logger.Error(err)
			return err
		}
	}

	return nil
}

func (c *Context) reset() {
	c.request = nil
	c.index = 0
	c.handlers = nil
	c.session = nil
}

func (c *Context) Header() *pkt.Header {
	return &c.request.Header
}

func (c *Context) ReadBody(val proto.Message) error {
	return c.request.ReadBody(val)
}

func (c *Context) Session() iface.ISession {
	if c.session == nil {
		server, _ := c.request.GetMeta(wire.MetaDestServer)
		c.session = &pkt.Session{
			ChannelId: c.request.ChannelId,
			GateId:    server.(string),
			Tags:      []string{"AutoGenerated"},
		}
	}
	return c.session
}