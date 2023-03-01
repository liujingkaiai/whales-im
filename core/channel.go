package core

import (
	"errors"
	"im/iface"
	"im/logger"
	"sync"
	"time"
)

type Channel struct {
	sync.Mutex
	id string
	iface.IConn
	meta      iface.IMeta
	writechan chan []byte
	once      sync.Once
	writewait time.Duration
	readwait  time.Duration
	closed    iface.IEvent
}

func NewChannel(id string, conn iface.IConn) iface.IChannel {
	log := logger.WithFields(logger.Fields{
		"module": "channel",
		"id":     id,
	})

	ch := &Channel{
		id:        id,
		IConn:     conn,
		writechan: make(chan []byte, 5),
		closed:    NewEvent(),
		writewait: 3 * time.Second,
		readwait:  3 * time.Second,
	}

	go func() {
		err := ch.wirteloop()
		if err != nil {
			log.Info(err)
		}
	}()
	return ch
}

func (ch *Channel) wirteloop() error {
	for {
		select {
		case payload := <-ch.writechan:
			err := ch.WriteFrame(iface.OpBinary, payload)
			if err != nil {
				return err
			}
			chanlen := len(ch.writechan)
			for i := 0; i < chanlen; i++ {
				payload = <-ch.writechan
				err := ch.WriteFrame(iface.OpBinary, payload)
				if err != nil {
					return err
				}
			}
			err = ch.IConn.Flush()
			if err != nil {
				return err
			}
		case <-ch.closed.Done():
			return nil
		}
	}
}

func (ch *Channel) ID() string {
	return ch.id
}

func (ch *Channel) GetMeta() iface.IMeta {
	return ch.meta
}

func (ch *Channel) Push(payload []byte) error {
	if len(payload) == 0 {
		return nil
	}
	return ch.WriteFrame(iface.OpBinary, payload)
}

func (ch *Channel) Close() error {
	ch.once.Do(func() {
		close(ch.writechan)
		ch.closed.Fire()
	})
	return nil
}

func (ch *Channel) SetWriteWait(t time.Duration) {
	ch.writewait = t
}

func (ch *Channel) SetReadWait(t time.Duration) {
	ch.readwait = t
}

func (ch *Channel) Readloop(lst iface.IMessageListener) error {
	ch.Lock()
	defer ch.Unlock()
	log := logger.WithFields(logger.Fields{
		"struct": "ChannelImpl",
		"func":   "Readloop",
		"id":     ch.id,
	})
	for {
		//_ = ch.SetReadDeadline(time.Now().Add(ch.readwait))

		frame, err := ch.ReadFrame()
		if err != nil {
			return err
		}

		if frame.GetOpCode() == iface.OpClose {
			return errors.New("remote side close the channe")
		}

		if frame.GetOpCode() == iface.OpPing {
			log.Trace("recv a ping; resp with a pong")
			_ = ch.WriteFrame(iface.OpPong, nil)
			continue
		}
		payload := frame.GetPayload()

		if len(payload) == 0 {
			continue
		}

		go lst.Receive(ch, payload)
	}
	return nil
}
