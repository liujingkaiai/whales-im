package iface

type Naming interface {

	//注册
	Register(service ServiceRegistration) error
	//注销
	Deregister(string) error
	//服务发现
	Find(serviceName string, name ...string) ([]ServiceRegistration, error)
	//订阅
	Subscribe(serviceName string, callback func(servicies []ServiceRegistration)) error
	//退订
	UnSubscribe(serviceName string) error
}

type ServiceRegistration interface {
	IService
	//ip or doamin
	PublicAddress() string
	PublicPort() int
	DialURL() string
	GetProtocol() string
	GetNamespace() string
	GetTags() []string
	// SetTags(tags []string)
	// SetMeta(meta map[string]string)
	String() string
}
