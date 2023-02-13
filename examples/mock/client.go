package mock

import (
	"context"
	"fmt"
	"im/iface"
	"im/logger"
	"im/websocket"
	"net"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

type ClientDemo struct {
}

func (c *ClientDemo) Start(userID string, protocol, addr string) {
	var cli iface.IClient
	if protocol == "ws" {
		cli = websocket.NewClient(userID, "client", websocket.ClientOptions{})
		// set dialer
		cli.SetDialer(&WebSocketDialer{})
	} else if protocol == "tcp" {
	}

	err := cli.Connect(addr)
	if err != nil {
		logger.Error(err)
	}

	logger.Info("connect server sucess!")
	count := 5
	go func() {
		for i := 0; i < 5; i++ {
			err := cli.Send([]byte("hello"))
			if err != nil {
				logger.Error(err)
				return
			}
			time.Sleep(10 * time.Millisecond)
			fmt.Println("send to server hello!")
		}
	}()

	// step4: 接收消息
	recv := 0
	for {
		frame, err := cli.Read()
		if err != nil {
			logger.Info(err)
			break
		}
		if frame.GetOpCode() != iface.OpBinary {
			continue
		}
		recv++
		logger.Warnf("%s receive message [%s]", cli.ID(), frame.GetPayload())
		if recv == count { // 接收完消息
			break
		}
	}
	//退出
	cli.Close()
}

type WebSocketDialer struct{}

func (d *WebSocketDialer) DialAndHandshake(ctx iface.DialerContext) (net.Conn, error) {
	conn, _, _, err := ws.Dial(context.TODO(), ctx.Address)
	if err != nil {
		return conn, err
	}

	err = wsutil.WriteClientBinary(conn, []byte(ctx.Id))
	if err != nil {
		conn.Close()
		return conn, err
	}

	return conn, nil
}

type TcpDialer struct{}

func (d *TcpDialer) DialAndHandshake(ctx iface.DialerContext) (net.Conn, error) {
	logger.Info("start dial: ", ctx.Address)

	conn, err := net.DialTimeout("tcp", ctx.Address, ctx.Timeout)
	if err != nil {
		return conn, err
	}

	return conn, nil
}
