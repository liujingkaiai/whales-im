package dialer

import (
	"bytes"
	"context"
	"fmt"
	"im/iface"
	"im/logger"
	"net"
	"time"

	"github.com/klintcheng/kim/wire"
	"github.com/klintcheng/kim/wire/pkt"
	"github.com/klintcheng/kim/wire/token"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

type ClientDialer struct {
}

func (d *ClientDialer) DialAndHandshake(ctx iface.DialerContext) (net.Conn, error) {
	logger.Info("DialAndHandshake called")
	//1.拨号
	conn, _, _, err := ws.Dial(context.Background(), ctx.Address)
	if err != nil {
		return conn, err
	}

	tk, err := token.Generate(token.DefaultSecret, &token.Token{
		Account: ctx.Id,
		App:     "kim",
		Exp:     time.Now().AddDate(0, 0, 1).Unix(),
	})
	if err != nil {
		return conn, err
	}
	//发送消息
	loginReq := pkt.New(wire.CommandLoginSignIn).WriteBody(&pkt.LoginReq{
		Token: tk,
	})
	err = wsutil.WriteClientBinary(conn, pkt.Marshal(loginReq))
	if err != nil {
		return nil, err
	}
	// wait resp
	logger.Info("waiting for login response")
	conn.SetReadDeadline(time.Now().Add(ctx.Timeout))
	frame, err := ws.ReadFrame(conn)
	if err != nil {
		fmt.Println("frame ", err)
		return nil, err
	}

	ack, err := pkt.MustReadLogicPkt(bytes.NewBuffer(frame.Payload))
	if err != nil {
		fmt.Println("ack ", err)
		return nil, err
	}

	if ack.Status != pkt.Status_Success {
		return nil, fmt.Errorf("login failed: %v", &ack.Header)
	}
	var resp = new(pkt.LoginResp)
	logger.Info("logined ", resp.GetChannelId())
	return conn, nil
}
