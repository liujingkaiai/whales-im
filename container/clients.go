package container

import (
	"fmt"
	"im/iface"
	"im/logger"
	"sync"
)

type Clients struct {
	clients sync.Map
}

func NewClients(num int) iface.IClientMap {
	return &Clients{clients: sync.Map{}}
}

// Add addChannel
func (ch *Clients) Add(client iface.IClient) {
	if client.ServiceID() == "" {
		logger.WithFields(logger.Fields{
			"module": "ClientsImpl",
		}).Error("client id is required")
	}
	ch.clients.Store(client.ServiceID(), client)
}

func (ch *Clients) Remove(id string) {
	ch.clients.Delete(id)
}

// Get Get
func (ch *Clients) Get(id string) (iface.IClient, bool) {
	if id == "" {
		logger.WithFields(logger.Fields{
			"module": "ClientsImpl",
		}).Error("client id is required")
	}
	val, ok := ch.clients.Load(id)
	if !ok {
		return nil, false
	}
	return val.(iface.IClient), true
}

func (ch *Clients) Servicies(kvs ...string) []iface.IService {
	kvLen := len(kvs)
	if kvLen != 0 && kvLen != 2 {
		return nil
	}

	arr := make([]iface.IService, 0)
	ch.clients.Range(func(key, value interface{}) bool {
		ser := value.(iface.IService)
		fmt.Println(ser.ServiceName())
		fmt.Println(ser.GetMeta())
		if kvLen > 0 && ser.GetMeta()[kvs[0]] != kvs[1] {
			return true
		}
		arr = append(arr, ser)
		return true
	})
	fmt.Println(arr)
	return arr
}
