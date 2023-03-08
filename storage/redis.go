package storage

import (
	"fmt"
	"im/iface"
	"time"

	redis "github.com/go-redis/redis/v7"
	"github.com/golang/protobuf/proto"
	"github.com/klintcheng/kim/wire/pkt"
)

const (
	LocationExpired = time.Hour * 48
)

type RedisStorage struct {
	cli *redis.Client
}

func NewRedisStoreage(cli *redis.Client) iface.ISessionStorage {
	return &RedisStorage{
		cli: cli,
	}
}

func (r *RedisStorage) Add(session *pkt.Session) error {
	loc := iface.Location{
		ChannelID: session.ChannelId,
		GateId:    session.GateId,
	}
	localKey := KeyLocation(session.Account, "")
	err := r.cli.Set(localKey, loc.Bytes(), LocationExpired).Err()
	if err != nil {
		return err
	}
	snKey := KeySession(session.ChannelId)
	buf, _ := proto.Marshal(session)
	return r.cli.Set(snKey, buf, LocationExpired).Err()
}

func (r *RedisStorage) Delete(account string, channelId string) error {
	locKey := KeyLocation(account, "")
	err := r.cli.Del(locKey).Err()
	if err != nil {
		return err
	}
	snKey := KeySession(channelId)
	return r.cli.Del(snKey).Err()
}

func (r *RedisStorage) Get(channelId string) (*pkt.Session, error) {
	snKey := KeySession(channelId)
	bts, err := r.cli.Get(snKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, iface.ErrSessionNil
		}
		return nil, err
	}
	var session pkt.Session
	proto.Unmarshal(bts, &session)
	return &session, err
}

func (r *RedisStorage) GetLocation(account string, device string) (*iface.Location, error) {
	key := KeyLocation(account, device)
	bts, err := r.cli.Get(key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, iface.ErrSessionNil
		}
		return nil, err
	}
	var loc iface.Location
	loc.Unmarshal(bts)
	return &loc, nil
}

func (r *RedisStorage) GetLocations(accounts ...string) ([]*iface.Location, error) {
	keys := KeyLocations(accounts...)
	list, err := r.cli.MGet(keys...).Result()
	if err != nil {
		return nil, err
	}
	result := make([]*iface.Location, 0, len(list))
	for _, l := range list {
		if l == nil {
			continue
		}
		var loc iface.Location
		loc.Unmarshal([]byte(l.(string)))
		result = append(result, &loc)
	}
	return result, nil
}

func KeySession(channel string) string {
	return fmt.Sprintf("login:sn:%s", channel)
}

func KeyLocation(account, device string) string {
	if device == "" {
		return fmt.Sprintf("login:loc:%s", account)
	}
	return fmt.Sprintf("login:loc:%s:%s", account, device)
}

func KeyLocations(accounts ...string) []string {
	arr := make([]string, len(accounts))
	for i, account := range accounts {
		arr[i] = KeyLocation(account, "")
	}
	return arr
}
