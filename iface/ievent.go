package iface

type IEvent interface {
	//执行一次
	Fire() bool
	//通知执行
	Done() <-chan struct{}
	//执行过没有
	HasFired() bool
}
