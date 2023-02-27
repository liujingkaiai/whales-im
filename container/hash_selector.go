package container

import (
	"im/iface"

	"github.com/klintcheng/kim/wire/pkt"
)

type HashSelector struct {
}

func (s *HashSelector) Lookup(header *pkt.Header, srvs []iface.IService) string {
	ll := len(srvs)
	code := HashCode(header.ChannelId)
	return srvs[code%ll].ServiceID()
}
