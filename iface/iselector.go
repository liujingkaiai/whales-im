package iface

import (
	"github.com/klintcheng/kim/wire/pkt"
)

//在消息上行时，从一批服务器中选出一个最适合的服务器
type Selector interface {
	Lookup(*pkt.Header, []IService) string
}
