package iface

type IClientMap interface {
	Add(client IClient)
	Remove(id string)
	Get(id string) (IClient, bool)

	Services(kvs ...string) []IService
}
