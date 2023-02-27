package websocket

import (
	"errors"
	"fmt"
	"im/iface"
	"im/logger"
	"net"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

// ClientOptions ClientOptions
type ClientOptions struct {
	Heartbeat time.Duration //登陆超时
	ReadWait  time.Duration //读超时
	WriteWait time.Duration //写超时
}

// Client is a websocket implement of the terminal
type Client struct {
	sync.Mutex
	iface.IDialer
	once    sync.Once
	id      string
	name    string
	conn    net.Conn
	state   int32
	options ClientOptions
	dc      *iface.DialerContext
	Meta    map[string]string
}

// NewClient NewClient
func NewClient(id, name string, opts ClientOptions) iface.IClient {
	if opts.WriteWait == 0 {
		opts.WriteWait = iface.DefaultWriteWait
	}
	if opts.ReadWait == 0 {
		opts.ReadWait = iface.DefaultReadWait
	}

	cli := &Client{
		id:      id,
		name:    name,
		options: opts,
		Meta:    make(map[string]string),
	}
	return cli
}

func NewClientWithProps(id, name string, meta map[string]string, opts ClientOptions) iface.IClient {
	if opts.WriteWait == 0 {
		opts.WriteWait = iface.DefaultWriteWait
	}
	if opts.ReadWait == 0 {
		opts.ReadWait = iface.DefaultReadWait
	}

	cli := &Client{
		id:      id,
		name:    name,
		options: opts,
		Meta:    meta,
	}
	return cli
}

func (c *Client) ServiceID() string {
	return c.id
}

func (c *Client) ServiceName() string {
	return c.name
}

func (c *Client) GetMeta() map[string]string {
	return c.Meta
}

func (c *Client) Connect(addr string) error {
	_, err := url.Parse(addr)
	if err != nil {
		return err
	}

	if !atomic.CompareAndSwapInt32(&c.state, 0, 1) {
		return fmt.Errorf("client has connected")
	}

	//拨号上网
	conn, err := c.DialAndHandshake(iface.DialerContext{
		Id:      c.id,
		Name:    c.name,
		Address: addr,
		Timeout: iface.DefaultLoginWait,
	})
	if err != nil {
		atomic.CompareAndSwapInt32(&c.state, 1, 0)
		return err
	}
	if conn == nil {
		return errors.New("conn is nil")
	}
	c.conn = conn
	if c.options.Heartbeat > 0 {
		go func() {
			err := c.heartloop(c.conn)
			if err != nil {
				logger.Error("heartbealoop stopped ", err)
			}
		}()
	}
	return nil
}

func (c *Client) Read() (iface.IFrame, error) {
	if c.conn == nil {
		return nil, errors.New("conn is nil ")
	}

	if c.options.ReadWait > 0 {
		_ = c.conn.SetReadDeadline(time.Now().Add(c.options.ReadWait))
	}

	frame, err := ws.ReadFrame(c.conn)
	if err != nil {
		return nil, err
	}
	if frame.Header.OpCode == ws.OpClose {
		return nil, errors.New("the connection is closed")
	}
	return &Frame{raw: frame}, nil
}

func (c *Client) Send(data []byte) error {
	if atomic.LoadInt32(&c.state) == 0 {
		return fmt.Errorf("conn is nil")
	}
	c.Lock()
	defer c.Unlock()
	if c.options.WriteWait > 0 {
		_ = c.conn.SetWriteDeadline(time.Now().Add(c.options.WriteWait))
	}
	return wsutil.WriteClientMessage(c.conn, ws.OpBinary, data)
}

func (c *Client) Close() {
	c.once.Do(func() {
		if c.conn == nil {
			return
		}

		wsutil.WriteClientMessage(c.conn, ws.OpClose, nil)
		c.conn.Close()
		atomic.CompareAndSwapInt32(&c.state, 1, 0)
	})
}

// SetDialer 设置握手逻辑
func (c *Client) SetDialer(dialer iface.IDialer) {
	c.IDialer = dialer
}

func (c *Client) heartloop(conn net.Conn) error {
	tick := time.NewTicker(c.options.Heartbeat)
	for range tick.C {
		if err := c.ping(conn); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) ping(conn net.Conn) error {
	c.Lock()
	defer c.Unlock()
	err := conn.SetWriteDeadline(time.Now().Add(c.options.WriteWait))
	if err != nil {
		return err
	}
	logger.Tracef("%s send ping to server", c.id)
	return wsutil.WriteClientMessage(conn, ws.OpCode(iface.OpPing), nil)
}
