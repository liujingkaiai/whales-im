package websocket

import (
	"context"
	"fmt"
	"im/core"
	"im/iface"
	"im/logger"
	"net/http"
	"sync"
	"time"

	"github.com/gobwas/ws"
	"github.com/pkg/errors"
	"github.com/segmentio/ksuid"
)

type ServerOptions struct {
	loginwait time.Duration //登陆超时
	readwait  time.Duration //读超时
	writewait time.Duration //写超时
}

type Server struct {
	listen string
	//服务注册
	iface.ServiceRegistration
	ChannelMap      iface.IChannelMap
	Acceptor        iface.IAcceptor
	MessageListener iface.IMessageListener
	Statelistener   iface.IStatelistener
	once            sync.Once
	options         ServerOptions
}

// NewServer NewServer
func NewServer(listen string, service iface.ServiceRegistration) iface.IServer {
	return &Server{
		listen:              listen,
		ServiceRegistration: service,
		options: ServerOptions{
			loginwait: iface.DefaultLoginWait,
			readwait:  iface.DefaultReadWait,
			writewait: time.Second * 10,
		},
	}
}

func (s *Server) Start() error {

	mux := http.NewServeMux()
	log := logger.WithFields(logger.Fields{
		"module": "ws.server",
		"listen": s.listen,
		"id":     s.ServiceID(),
	})
	if s.Acceptor == nil {
		s.Acceptor = new(defaultAcceptor)
	}
	if s.Statelistener == nil {
		return fmt.Errorf("StateListener is nil")
	}
	if s.ChannelMap == nil {
		s.ChannelMap = core.NewChannels(100)
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		raw, _, _, err := ws.UpgradeHTTP(r, w)
		if err != nil {
			return
		}

		//包装conn
		conn := NewConn(raw)
		//鉴权
		id, err := s.Acceptor.Accept(conn, s.options.loginwait)
		if err != nil {
			fmt.Println(err)
			_ = conn.WriteFrame(iface.OpClose, []byte(err.Error()))
			conn.Close()
			return
		}

		if _, ok := s.ChannelMap.Get(id); ok {
			log.Warnf("channel %s existed", id)
			_ = conn.WriteFrame(iface.OpClose, []byte("channelId is repeated"))
			conn.Close()
			return
		}

		// step 4
		channel := core.NewChannel(id, conn)
		channel.SetWriteWait(s.options.writewait)
		channel.SetReadWait(s.options.readwait)
		s.ChannelMap.Add(channel)
		go func(channel iface.IChannel) {

			err := channel.Readloop(s.MessageListener)
			if err != nil {
				log.Info(err)
			}
			s.ChannelMap.Remove(channel.ID())
			err = s.Statelistener.Disconnect(channel.ID())
			if err != nil {
				log.Warn(err)
			}
			channel.Close()
		}(channel)
	})
	return http.ListenAndServe(s.listen, mux)
}

func (s *Server) Push(id string, data []byte) error {
	ch, ok := s.ChannelMap.Get(id)
	if !ok {
		return errors.Errorf("push to channel [ID]:%s , channel not found", id)
	}
	return ch.Push(data)
}

func (s *Server) Shutdown(ctx context.Context) error {
	log := logger.WithFields(logger.Fields{
		"module": "ws.server",
		"id":     s.ServiceID(),
	})

	s.once.Do(func() {
		defer func() {
			log.Infoln("shutdown")
		}()
		channels := s.ChannelMap.All()
		for _, ch := range channels {
			ch.Close()
			select {
			case <-ctx.Done():
				return
			default:
			}
		}
	})

	return nil
}

// SetAcceptor SetAcceptor
func (s *Server) SetAcceptor(acceptor iface.IAcceptor) {
	s.Acceptor = acceptor
}

// SetMessageListener SetMessageListener
func (s *Server) SetMessageListener(listener iface.IMessageListener) {
	s.MessageListener = listener
}

// SetStateListener SetStateListener
func (s *Server) SetStateListener(listener iface.IStatelistener) {
	s.Statelistener = listener
}

// SetChannels SetChannels
func (s *Server) SetChannelMap(channels iface.IChannelMap) {
	s.ChannelMap = channels
}

// SetReadWait set read wait duration
func (s *Server) SetReadWait(readwait time.Duration) {
	s.options.readwait = readwait
}

func resp(w http.ResponseWriter, code int, body string) {
	w.WriteHeader(code)
	if body != "" {
		_, _ = w.Write([]byte(body))
	}
	logger.Warnf("response with code:%d %s", code, body)
}

type defaultAcceptor struct {
}

// Accept defaultAcceptor
func (a *defaultAcceptor) Accept(conn iface.IConn, timeout time.Duration) (string, error) {
	return ksuid.New().String(), nil
}
