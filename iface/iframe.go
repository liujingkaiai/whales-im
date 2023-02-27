package iface

//拆包解包
type IFrame interface {
	SetOpCode(OpCode)
	GetOpCode() OpCode
	//封包
	SetPayload([]byte)
	//拆包
	GetPayload() []byte
}
