package container

import (
	"im/iface"
	"im/logger"
	"sync"
)

const (
	stateUninitialized = iota
	stateInitialized
	stateStarted
	stateClosed
)

const (
	StateYoung = "young"
	StateAdult = "adult"
)

const (
	KeyServiceState = "service_state"
)

type Container struct {
	sync.RWMutex
	Name       iface.Naming
	Srv        iface.IServer
	state      uint32
	srvclients map[string]iface.IClientMap
	selector   iface.Selector
	dialer     iface.IDialer
	deps       map[string]struct{}
}

var log = logger.WithField("module", "container")

// Default Container
var c = &Container{
	state:    0,
	selector: &HashSelector{},
	deps:     make(map[string]struct{}),
}

// Default Default
func Default() *Container {
	return c
}
