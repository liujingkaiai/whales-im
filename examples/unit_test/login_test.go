package unittest_test

import (
	"im/examples/dialer"
	"im/iface"
	"im/websocket"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func login(account string) (iface.IClient, error) {
	cli := websocket.NewClient(account, "unittest", websocket.ClientOptions{})

	cli.SetDialer(&dialer.ClientDialer{})
	err := cli.Connect("ws://localhost:8000")
	if err != nil {
		return nil, err
	}
	return cli, nil
}

func Test_login(t *testing.T) {
	cli, err := login("test1")
	assert.Nil(t, err)
	time.Sleep(time.Second * 3)
	cli.Close()
}
