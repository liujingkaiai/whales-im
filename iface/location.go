package iface

import (
	"bytes"
	"errors"
	"im/wire/endian"
)

type Location struct {
	ChannelID string //网关中的channelID
	GateId    string //网关ID
}

func (loc *Location) Bytes() []byte {
	if loc == nil {
		return []byte{}
	}
	buf := &bytes.Buffer{}
	endian.WriteShortBytes(buf, []byte(loc.ChannelID))
	endian.WriteShortBytes(buf, []byte(loc.GateId))
	return buf.Bytes()
}

func (loc *Location) Unmarshal(data []byte) (err error) {
	if len(data) == 0 {
		return errors.New("data is empty")
	}
	buf := bytes.NewBuffer(data)
	loc.ChannelID, err = endian.ReadShortString(buf)
	if err != nil {
		return
	}
	loc.GateId, err = endian.ReadShortString(buf)
	if err != nil {
		return
	}
	return
}
