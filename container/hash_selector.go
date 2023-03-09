package container

import (
	"fmt"
	"im/iface"

	"github.com/klintcheng/kim/wire/pkt"
)

type HashSelector struct {
}

func (s *HashSelector) Lookup(header *pkt.Header, srvs []iface.IService) string {
	ll := len(srvs)
	code := HashCode(header.ChannelId)
	fmt.Printf("header.ChannelId:%v code:%v %% len(srvs):%v = %v\n", header.ChannelId, code, ll, code%ll)
	return srvs[code%ll].ServiceID()
}
