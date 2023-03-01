package serv

import (
	"im/iface"
	"im/logger"
	"im/tcp"
	"net"

	"github.com/klintcheng/kim/wire/pkt"
	"google.golang.org/protobuf/proto"
)

type TcpDialer struct {
	ServiceID string
}

func NewDialer(serviceId string) iface.IDialer {
	return &TcpDialer{
		ServiceID: serviceId,
	}
}

// DialAndHandshake(context.Context, string) (net.Conn, error)
func (d *TcpDialer) DialAndHandshake(ctx iface.DialerContext) (net.Conn, error) {
	// 1. 拨号建立连接
	conn, err := net.DialTimeout("tcp", ctx.Address, ctx.Timeout)
	if err != nil {
		return nil, err
	}
	req := &pkt.InnerHandshakeReq{
		ServiceId: d.ServiceID,
	}
	logger.Infof("send req %v", req)
	// 2. 把自己的ServiceId发送给对方
	bts, _ := proto.Marshal(req)
	err = tcp.WriteFrame(conn, iface.OpBinary, bts)
	if err != nil {
		return nil, err
	}
	return conn, nil
}
