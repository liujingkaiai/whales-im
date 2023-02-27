package iface

//消息监听器
type IMessageListener interface {
	Receive(IAgent, []byte)
}

//发送方
type IAgent interface {
	//返回连接的channelid
	ID() string
	//返回channelid
	Push([]byte) error
}
