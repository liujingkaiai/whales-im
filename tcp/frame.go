package tcp

import "im/iface"

type Frame struct {
	OpCode  iface.OpCode
	Payload []byte
}

func (f *Frame) SetOpCode(opcode iface.OpCode) {
	f.OpCode = opcode
}

func (f *Frame) GetOpCode() iface.OpCode {
	return f.OpCode
}

func (f *Frame) SetPayload(payload []byte) {
	f.Payload = payload
}

func (f *Frame) GetPayload() []byte {
	return f.Payload
}
