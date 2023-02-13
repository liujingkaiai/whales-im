package iface

type IChannelMap interface {
	Add(channel IChannel)
	Remove(string)
	Get(string) (IChannel, bool)
	All() []IChannel
}
