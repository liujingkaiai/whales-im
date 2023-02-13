package tcp

import (
	"errors"
	"fmt"
	"im/iface"
	"im/logger"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

type ClientOptions struct {
	Heartbeat time.Duration
	ReadWait  time.Duration
	WriteWait time.Duration
}

type Client struct {
	sync.Mutex
	iface.IDialer
	once    sync.Once
	id      string
	name    string
	conn    iface.IConn
	state   int32
	options ClientOptions
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
	}
	return cli
}

func (c *Client) Connect(addr string) error {
	if _, err := url.Parse(addr); err != nil {
		return err
	}

	if !atomic.CompareAndSwapInt32(&c.state, 0, 1) {
		return fmt.Errorf("connection is connected")
	}

	rawconn, err := c.DialAndHandshake(iface.DialerContext{
		Id:      c.id,
		Name:    c.name,
		Address: addr,
		Timeout: iface.DefaultLoginWait,
	})

	if err != nil {
		atomic.CompareAndSwapInt32(&c.state, 1, 0)
		return err
	}

	if rawconn == nil {
		return fmt.Errorf("conn is nil")
	}

	c.conn = NewTcpConn(rawconn)

	if c.options.Heartbeat > 0 {
		//心跳处理
		go func() {
			err := c.heartbealoop()
			if err != nil {
				logger.WithField("module", "tcp.client").Warn("heartbealoop stopped - ", err)
			}
		}()
	}
	return nil
}

func (c *Client) Read() (iface.IFrame, error) {
	if c.conn == nil {
		return nil, errors.New("conn is nil")
	}

	if c.options.ReadWait > 0 {
		c.conn.SetReadDeadline(time.Now().Add(c.options.ReadWait))
	}
	frame, err := c.conn.ReadFrame()
	if err != nil {
		return frame, err
	}

	if frame.GetOpCode() == iface.OpClose {
		return frame, errors.New("conn is closed")
	}
	return frame, nil
}

func (c *Client) Send(payload []byte) error {
	if atomic.LoadInt32(&c.state) == 0 {
		return fmt.Errorf("conn is nil")
	}
	c.Lock()
	defer c.Unlock()
	err := c.conn.SetWriteDeadline(time.Now().Add(c.options.WriteWait))
	if err != nil {
		return err
	}
	return c.conn.WriteFrame(iface.OpBinary, payload)
}

func (c *Client) SetDialer(dailer iface.IDialer) {
	c.IDialer = dailer
}

func (c *Client) ID() string {
	return c.id
}

func (c *Client) Name() string {
	return c.name
}

func (c *Client) Close() {
	c.once.Do(func() {
		if c.conn == nil {
			return
		}
		_ = c.conn.WriteFrame(iface.OpClose, nil)
		c.conn.Close()
		atomic.CompareAndSwapInt32(&c.state, 1, 0)
	})
}

func (c *Client) heartbealoop() error {
	tick := time.NewTicker(c.options.Heartbeat)
	for range tick.C {
		if err := c.ping(); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) ping() error {
	logger.WithField("module", "tcp.client").Tracef("%s send ping to server", c.id)
	err := c.conn.SetWriteDeadline(time.Now().Add(c.options.WriteWait))
	if err != nil {
		return err
	}
	return c.conn.WriteFrame(iface.OpPing, nil)
}
