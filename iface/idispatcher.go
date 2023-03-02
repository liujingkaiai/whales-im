package iface

import (
	"github.com/klintcheng/kim/wire/pkt"
)

type Dispatcher interface {
	Push(gateway string, channels []string, p *pkt.LogicPkt) error
}
