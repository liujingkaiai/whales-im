package container

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"im/iface"
	"im/logger"
	"im/tcp"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
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

// 初始化
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

// 设置selector
func SetSelector(selector iface.Selector) {
	c.selector = selector
}

func SetServiceNaming(nm iface.Naming) {
	c.Name = nm
}

// 启动容器
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
			err := connectToService(service)
			if err != nil {
				log.Errorln(err)
			}
		}(serive)
	}

	//3.服务注册
	if c.Srv.PublicAddress() != "" && c.Srv.PublicPort() != 0 {
		err := c.Name.Register(c.Srv)
		if err != nil {
			log.Warn(err)
		}
	}
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	log.Infoln("shutdown ", <-c)

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

// 消息通过网关服务器推送到channel中
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

// 下行消息-------->push到网关服务 [指tcp/websocket服务]
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
		return nil, fmt.Errorf("service %s not found", serviceName)
	}
	// 只获取状态为StateAdult的服务
	srvs := clients.Servicies(KeyServiceState, StateAdult)
	if len(srvs) == 0 {
		return nil, fmt.Errorf("no services found for %s", serviceName)
	}
	id := selector.Lookup(header, srvs)
	if cli, ok := clients.Get(id); ok {
		return cli, nil
	}
	fmt.Println("get cli")
	return nil, fmt.Errorf("no client found")
}

func connectToService(serviceName string) error {
	clients := NewClients(10)
	if clients == nil {
		return errors.New("client map length is 0")
	}
	c.srvclients[serviceName] = clients
	// 1.观察服务新增
	delay := time.Second * 10
	err := c.Name.Subscribe(serviceName, func(servicies []iface.ServiceRegistration) {
		for _, service := range servicies {
			if _, ok := clients.Get(service.ServiceID()); ok {
				continue
			}
			log.WithField("func", "connectToService").Infof("Watch a new service: %v", service)
			service.GetMeta()[KeyServiceState] = StateYoung

			go func(service iface.ServiceRegistration) {
				time.Sleep(delay)
				service.GetMeta()[KeyServiceState] = StateAdult
				fmt.Println(c.srvclients[serviceName])
			}(service)

			_, err := buildClient(clients, service)
			if err != nil {
				logger.Warn(err)
			}
		}
	})
	if err != nil {
		return err
	}

	// 2. 再查询已经存在的服务
	services, err := c.Name.Find(serviceName)
	if err != nil {
		return err
	}
	log.Info("find service ", services)
	for _, service := range services {
		// 标记为StateAdult
		service.GetMeta()[KeyServiceState] = StateAdult
		_, err := buildClient(clients, service)
		if err != nil {
			logger.Warn(err)
		}
	}
	return nil
}

func buildClient(clients iface.IClientMap, service iface.ServiceRegistration) (iface.IClient, error) {
	c.Lock()
	defer c.Unlock()
	var (
		id   = service.ServiceID()
		name = service.ServiceName()
		meta = service.GetMeta()
	)
	if _, ok := clients.Get(id); ok {
		return nil, nil
	}
	//2.服务之间只能用tcp
	if service.GetProtocol() != string(wire.ProtocolTCP) {
		return nil, fmt.Errorf("unexpected service Protocol: %s", service.GetProtocol())
	}

	// // 3. 构建客户端并建立连接
	cli := tcp.NewClientWithProps(id, name, meta, tcp.ClientOptions{
		Heartbeat: time.Minute,
		ReadWait:  time.Minute * 3,
		WriteWait: time.Second * 10,
	})
	if c.dialer == nil {
		return nil, fmt.Errorf("dialer is nil")
	}

	cli.SetDialer(c.dialer)

	err := cli.Connect(service.DialURL())
	if err != nil {
		return nil, err
	}
	//读取消息
	go func(cli iface.IClient) {
		err := readloop(cli)
		if err != nil {
			log.Debug(err)
		}
		clients.Remove(id)
		cli.Close()
	}(cli)
	clients.Add(cli)
	return cli, nil
}

func readloop(cli iface.IClient) error {
	log := logger.WithFields(logger.Fields{
		"module": "container",
		"func":   "readLoop",
	})
	log.Infof("readLoop started of %s %s", cli.ServiceID(), cli.ServiceName())
	for {
		frame, err := cli.Read()
		if err != nil {
			return err
		}

		if frame.GetOpCode() != iface.OpBinary {
			continue
		}

		buf := bytes.NewBuffer(frame.GetPayload())

		packet, err := pkt.MustReadLogicPkt(buf)
		if err != nil {
			log.Info(err)
			continue
		}

		err = pushMessage(packet)
		if err != nil {
			log.Info(err)
		}
	}
	return nil
}
