package iface

import (
	"github.com/klintcheng/kim/wire/pkt"
)

type Selector interface {
	Lookup(*pkt.Header, []IService) string
}
