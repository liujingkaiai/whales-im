package core

import (
	"im/iface"
	"im/logger"
	"sync"
)

type ChannelMannager struct {
	channels sync.Map
}

func NewChannels(num int) iface.IChannelMap {
	return &ChannelMannager{
		channels: sync.Map{},
	}
}

func (cm *ChannelMannager) Add(channel iface.IChannel) {
	cm.checkID(channel.ID())
	cm.channels.Store(channel.ID(), channel)
}

func (cm *ChannelMannager) Remove(id string) {
	cm.checkID(id)
	cm.channels.Delete(id)
}

func (cm *ChannelMannager) Get(id string) (iface.IChannel, bool) {
	channel, ok := cm.channels.Load(id)
	if !ok {
		return nil, ok
	}

	if ch, ok := channel.(iface.IChannel); ok {
		return ch, ok
	}
	return nil, false
}

func (cm *ChannelMannager) checkID(channel string) {
	if len(channel) == 0 {
		logger.WithFields(logger.Fields{
			"module": "ChannelsImpl",
		}).Error("channel id is required")
	}
}

// All return channels
func (ch *ChannelMannager) All() []iface.IChannel {
	arr := make([]iface.IChannel, 0)
	ch.channels.Range(func(key, val interface{}) bool {
		arr = append(arr, val.(iface.IChannel))
		return true
	})
	return arr
}
