package mock

import (
	"errors"
	"im/iface"
	"im/logger"
	"im/naming"
	"im/websocket"
	"time"
)

type ServerDemo struct{}

func (s *ServerDemo) Start(id, protocol, addr string) {
	var srv iface.IServer
	service := naming.DefaultService{
		Id:       id,
		Protocol: protocol,
	}

	if protocol == "ws" {
		srv = websocket.NewServer(addr, &service)
	}
	handler := &ServerHandler{}

	srv.SetReadWait(time.Minute)
	srv.SetAcceptor(handler)
	srv.SetMessageListener(handler)
	srv.SetStateListener(handler)
	err := srv.Start()
	if err != nil {
		panic(err)
	}
}

type ServerHandler struct{}

func (h *ServerHandler) Accept(conn iface.IConn, timeout time.Duration) (string, error) {
	//第一次发包  发的是鉴权包
	frame, err := conn.ReadFrame()
	if err != nil {
		return "", err
	}
	logger.Info("recv", frame.GetOpCode())
	// 2. 解析：数据包内容就是userId
	userID := string(frame.GetPayload())
	// 3. 鉴权：这里只是为了示例做一个fake验证，非空
	if userID == "" {
		return "", errors.New("user id is invalid")
	}
	return userID, nil
}

// Receive default listener
func (h *ServerHandler) Receive(ag iface.IAgent, payload []byte) {
	ack := string(payload)
	_ = ag.Push([]byte(ack))
}

// Disconnect default listener
func (h *ServerHandler) Disconnect(id string) error {
	logger.Warnf("disconnect %s", id)
	return nil
}
