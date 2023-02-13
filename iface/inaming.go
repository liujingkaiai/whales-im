package iface

type Naming interface {
	Find(serviceName string)
	Remove(serviceName, serviceID string) error
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
