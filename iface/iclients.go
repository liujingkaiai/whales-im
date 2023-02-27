package iface

type IClientMap interface {
	Add(client IClient)
	Remove(id string)
	Get(id string) (IClient, bool)

	Servicies(kvs ...string) []IService
}
