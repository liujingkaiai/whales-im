package container

import (
	"context"
	"errors"
	"fmt"
	"im/iface"
	"im/logger"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/klintcheng/kim/wire"
	"github.com/klintcheng/kim/wire/pkt"
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

//初始化
func Init(srv iface.IServer, deps ...string) error {
	//检测是否初始化
	if !atomic.CompareAndSwapUint32(&c.state, stateUninitialized, stateInitialized) {
		return errors.New("has Initialized")
	}
	c.Srv = srv

	for _, dep := range deps {
		if _, ok := c.deps[dep]; ok {
			continue
		}
		c.deps[dep] = struct{}{}
	}
	log.WithField("func", "Init").Infof("srv %s:%s - deps %v", srv.ServiceID(), srv.ServiceName(), c.deps)
	c.srvclients = make(map[string]iface.IClientMap, len(deps))
	return nil
}

func SetDialer(dialer iface.IDialer) {
	c.dialer = dialer
}

//设置selector
func SetSelector(selector iface.Selector) {
	c.selector = selector
}

//启动容器
func Start() error {
	if c.Name == nil {
		return fmt.Errorf("naming is nil")
	}

	if !atomic.CompareAndSwapUint32(&c.state, stateInitialized, stateStarted) {
		return errors.New("has started")
	}

	//1.启动服务
	go func(srv iface.IServer) {
		err := srv.Start()
		if err != nil {
			log.Fatalln(err)
		}
	}(c.Srv)

	//2.与依赖服务简历连接
	for serive := range c.deps {
		go func(service string) {

		}(serive)
	}

	return shutdown()
}

func shutdown() error {
	if !atomic.CompareAndSwapUint32(&c.state, stateStarted, stateClosed) {
		return errors.New("has closed")
	}
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*10)
	defer cancel()

	//关闭 container server
	err := c.Srv.Shutdown(ctx)
	if err != nil {
		log.Fatal(err)
	}

	//从注册中心销毁服务
	err = c.Name.Deregister(c.Srv.ServiceID())
	if err != nil {
		log.Warn(err)
		return err
	}
	//退订服务变更
	for dep := range c.deps {
		_ = c.Name.Deregister(dep)
	}
	log.Info("shutdown")
	return nil
}

//消息通过网关服务器推送到channel中
func pushMessage(packet *pkt.LogicPkt) error {
	server, _ := packet.GetMeta(wire.MetaDestServer)
	if server != c.Srv.ServiceID() {
		return fmt.Errorf("dest_server is incorrect, %s != %s", server, c.Srv.ServiceID())
	}
	channels, ok := packet.GetMeta(wire.MetaDestChannels)
	if !ok {
		return fmt.Errorf("dest_channels is nil")
	}

	channelIds := strings.Split(channels.(string), ",")
	packet.DelMeta(wire.MetaDestServer)
	packet.DelMeta(wire.MetaDestChannels)
	payload := pkt.Marshal(packet)
	log.Debugf("Push to %v %v", channelIds, packet)

	for _, channel := range channelIds {
		err := c.Srv.Push(channel, payload)
		if err != nil {
			log.Debug(err)
		}
	}
	return nil
}

//下行消息-------->push到网关服务
func Push(server string, p *pkt.LogicPkt) error {
	p.AddStringMeta(wire.MetaDestServer, server)
	return c.Srv.Push(server, pkt.Marshal(p))
}

// Forward message to service
func Forward(serviceName string, packet *pkt.LogicPkt) error {
	if packet == nil {
		return errors.New("packet is nil")
	}
	if packet.Command == "" {
		return errors.New("command is empty in packet")
	}
	if packet.ChannelId == "" {
		return errors.New("ChannelId is empty in packet")
	}
	return ForwardWithSelector(serviceName, packet, c.selector)
}

// ForwardWithSelector forward data to the specified node of service which is chosen by selector
func ForwardWithSelector(serviceName string, packet *pkt.LogicPkt, selector iface.Selector) error {
	cli, err := lookup(serviceName, &packet.Header, selector)
	if err != nil {
		return err
	}
	// add a tag in packet
	packet.AddStringMeta(wire.MetaDestServer, c.Srv.ServiceID())
	log.Debugf("forward message to %v with %s", cli.ServiceID(), &packet.Header)
	return cli.Send(pkt.Marshal(packet))
}

func lookup(serviceName string, header *pkt.Header, selector iface.Selector) (iface.IClient, error) {
	clients, ok := c.srvclients[serviceName]
	if !ok {
		return nil, fmt.Errorf("client %s is not regsiter in serve", serviceName)
	}

	srvs := clients.Servicies(KeyServiceState, StateAdult)
	if len(srvs) == 0 {
		return nil, fmt.Errorf("no servicies for %s", serviceName)
	}

	id := selector.Lookup(header, srvs)
	if cli, ok := clients.Get(id); ok {
		return cli, nil
	}
	return nil, fmt.Errorf("no client found")
}

func connectToService(serviceName string) error {
	clients := NewClients(10)
	if clients == nil {
		return errors.New("client map length is 0")
	}
	return nil
}
