package tcp

import (
	"context"
	"errors"
	"fmt"
	"im/core"
	"im/iface"
	"im/logger"
	"net"
	"sync"
	"time"

	"github.com/segmentio/ksuid"
)

type ServerOption struct {
	loginwait time.Duration
	readwait  time.Duration
	writewait time.Duration
}

//tcp server
type Server struct {
	listen string
	iface.ServiceRegistration
	ChannelMap      iface.IChannelMap
	Acceptor        iface.IAcceptor
	MessageListener iface.IMessageListener
	StateListener   iface.IStatelistener
	once            sync.Once
	options         ServerOption
	quit            iface.IEvent
}

func NewServer(addr string, service iface.ServiceRegistration) iface.IServer {
	return &Server{
		listen:              addr,
		ServiceRegistration: service,
		ChannelMap:          core.NewChannels(100),
		quit:                core.NewEvent(),
		options: ServerOption{
			loginwait: iface.DefaultLoginWait,
			readwait:  iface.DefaultReadWait,
			writewait: iface.DefaultWriteWait,
		},
	}
}

//启动服务
func (srv *Server) Start() error {
	log := logger.WithFields(logger.Fields{
		"module": "tcp.server",
		"listen": srv.listen,
		"id":     srv.ServiceID(),
	})

	if srv.StateListener == nil {
		return errors.New("statelistener is nil")
	}

	if srv.Acceptor == nil {
		srv.Acceptor = new(defaultAcceptor)
	}

	lis, err := net.Listen("tcp", srv.listen)
	if err != nil {
		return err
	}

	log.Infof("tcp server started on port:%s\n", srv.listen)
	for {
		rawconn, err := lis.Accept()
		if err != nil {
			rawconn.Close()
			log.Warn(err)
			continue
		}

		go func(rawconn net.Conn) {
			conn := NewTcpConn(rawconn)
			id, err := srv.Acceptor.Accept(conn, srv.options.loginwait)
			if err != nil {
				_ = conn.WriteFrame(iface.OpClose, []byte(err.Error()))
				conn.Close()
				return
			}

			if _, ok := srv.ChannelMap.Get(id); ok {
				_ = conn.WriteFrame(iface.OpClose, []byte("channel id is connected"))
				conn.Close()
				return
			}

			channel := core.NewChannel(id, conn)
			channel.SetReadWait(srv.options.readwait)
			channel.SetWriteWait(srv.options.writewait)

			srv.ChannelMap.Add(channel)
			log.Info("accept ", channel)
			err = channel.Readloop(srv.MessageListener)
			if err != nil {
				srv.ChannelMap.Remove(channel.ID())
			}

			_ = srv.StateListener.Disconnect(channel.ID())
			channel.Close()
		}(rawconn)

		select {
		case <-srv.quit.Done():
			return fmt.Errorf("listen exited")
		default:
		}
	}
	return nil
}

//根据id给连接发送消息
func (srv *Server) Push(id string, payload []byte) error {
	channel, ok := srv.ChannelMap.Get(id)
	if !ok {
		return errors.New("channel:" + channel.ID() + " not found")
	}
	return channel.Push(payload)
}

//停止服务
func (s *Server) Shutdown(ctx context.Context) error {
	log := logger.WithFields(logger.Fields{
		"module": "tcp.server",
		"id":     s.ServiceID(),
	})

	s.once.Do(func() {
		defer func() {
			log.Infoln("shutdown")
		}()

		channels := s.ChannelMap.All()
		for _, channel := range channels {
			channel.Close()
			select {
			case <-ctx.Done():
				return
			default:
				continue
			}
		}
	})
	return nil
}

func (srv *Server) SetAcceptor(acceptor iface.IAcceptor) {
	srv.Acceptor = acceptor
}

func (srv *Server) SetMessageListener(messagelistener iface.IMessageListener) {
	srv.MessageListener = messagelistener
}

func (srv *Server) SetStateListener(statelistener iface.IStatelistener) {
	srv.StateListener = statelistener
}

func (srv *Server) SetReadWait(readwait time.Duration) {
	srv.options.readwait = readwait
}

func (srv *Server) SetChannelMap(channelMap iface.IChannelMap) {
	srv.ChannelMap = channelMap
}

type defaultAcceptor struct {
}

func (acp *defaultAcceptor) Accept(conn iface.IConn, readwait time.Duration) (string, error) {
	return ksuid.New().String(), nil
}
