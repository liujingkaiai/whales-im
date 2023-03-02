package iface

import (
	"errors"

	"github.com/klintcheng/kim/wire/pkt"
)

var ErrSessionNil = errors.New("err:session nil")

type ISessionStorage interface {
	Add(session *pkt.Session) error
	Delete(account string, channleID string) error
	Get(string) (*pkt.Session, error)
	GetLocations(...string) ([]*Location, error)
	GetLocation(string) (*Location, error)
}
