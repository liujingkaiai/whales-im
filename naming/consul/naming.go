package consul

import (
	"errors"
	"fmt"
	"im/iface"
	"im/logger"
	"im/naming"
	"sync"
	"time"

	"github.com/hashicorp/consul/api"
)

const (
	KeyProtocol  = "protocol"
	KeyHealthURL = "health_url"
)

const (
	HealthPassing  = "passing"
	HealthWarning  = "warning"
	HealthCritical = "critical"
	HealthMaint    = "maintenance"
)

type Watch struct {
	Service   string
	Callback  func([]iface.ServiceRegistration)
	WaitIndex uint64
	Quit      chan struct{}
}

type Naming struct {
	sync.RWMutex
	cli     *api.Client
	watches map[string]*Watch
}

func NewNaming(consulUrl string) (iface.Naming, error) {
	conf := api.DefaultConfig()
	conf.Address = consulUrl
	cli, err := api.NewClient(conf)
	if err != nil {
		return nil, err
	}
	nami := &Naming{
		cli:     cli,
		watches: make(map[string]*Watch),
	}
	return nami, nil
}

func (n *Naming) Register(s iface.ServiceRegistration) error {
	reg := &api.AgentServiceRegistration{
		ID:      s.ServiceID(),
		Name:    s.ServiceName(),
		Address: s.PublicAddress(),
		Port:    s.PublicPort(),
		Tags:    s.GetTags(),
		Meta:    s.GetMeta(),
	}

	if reg.Meta == nil {
		reg.Meta = make(map[string]string)
	}
	reg.Meta[KeyProtocol] = s.GetProtocol()
	// consul健康检查
	healthURL := s.GetMeta()[KeyHealthURL]
	if healthURL != "" {
		check := new(api.AgentServiceCheck)
		check.CheckID = fmt.Sprintf("%s_normal", s.ServiceID())
		check.HTTP = healthURL
		check.Timeout = "1s" // http timeout
		check.Interval = "10s"
		check.DeregisterCriticalServiceAfter = "20s"
		reg.Check = check
	}
	err := n.cli.Agent().ServiceRegister(reg)
	return err
}

func (n *Naming) Deregister(serviceID string) error {
	return n.cli.Agent().ServiceDeregister(serviceID)
}

func (n *Naming) Find(name string, tags ...string) ([]iface.ServiceRegistration, error) {
	services, _, err := n.load(name, 0, tags...)
	return services, err
}

func (n *Naming) load(name string, waitIndex uint64, tags ...string) ([]iface.ServiceRegistration, *api.QueryMeta, error) {
	opts := &api.QueryOptions{
		UseCache:  true,
		MaxAge:    time.Minute,
		WaitIndex: waitIndex,
	}
	catalogServices, meta, err := n.cli.Catalog().ServiceMultipleTags(name, tags, opts)
	if err != nil {
		return nil, meta, err
	}

	services := make([]iface.ServiceRegistration, len(catalogServices))
	for i, s := range catalogServices {
		if s.Checks.AggregatedStatus() != api.HealthPassing {
			logger.Debugf("load service: id:%s name:%s %s:%d Status:%s", s.ServiceID, s.ServiceName, s.ServiceAddress, s.ServicePort, s.Checks.AggregatedStatus())
			continue
		}
		services[i] = &naming.DefaultService{
			Id:       s.ServiceID,
			Name:     s.ServiceName,
			Address:  s.ServiceAddress,
			Port:     s.ServicePort,
			Protocol: s.ServiceMeta[KeyProtocol],
			Tags:     s.ServiceTags,
			Meta:     s.ServiceMeta,
		}
	}
	logger.Debugf("load service: %v, meta:%v", services, meta)
	return services, meta, nil
}

func (n *Naming) Subscribe(serviceName string, callback func([]iface.ServiceRegistration)) error {
	n.Lock()
	defer n.Unlock()

	if _, ok := n.watches[serviceName]; ok {
		return errors.New("serviceName has already been registered")
	}

	w := &Watch{
		Service:  serviceName,
		Callback: callback,
		Quit:     make(chan struct{}),
	}
	n.watches[serviceName] = w
	n.watch(w)
	return nil
}

func (n *Naming) UnSubscribe(serviceName string) error {
	n.Lock()
	defer n.Unlock()
	wh, ok := n.watches[serviceName]
	if ok {
		delete(n.watches, serviceName)
		close(wh.Quit)
	}
	return nil
}

func (n *Naming) watch(wh *Watch) {
	stopped := false
	var doWatch = func(service string, callback func([]iface.ServiceRegistration)) {
		services, meta, err := n.load(service, wh.WaitIndex) // <-- blocking untill services has changed
		if err != nil {
			logger.Warn(err)
			return
		}
		select {
		case <-wh.Quit:
			stopped = true
			logger.Infof("watch %s stopped", wh.Service)
			return
		default:
		}
		wh.WaitIndex = meta.LastIndex
		if callback != nil {
			callback(services)
		}
	}

	doWatch(wh.Service, nil)
	for !stopped {
		doWatch(wh.Service, wh.Callback)
	}
}
