package client

import (
	"errors"
	"net/rpc"
	"strconv"

	"github.com/SugierPawel/player/rpc/core"
)

type Client struct {
	Port   int
	client *rpc.Client
}

func (c *Client) Init() (err error) {
	if c.Port == 0 {
		err = errors.New("client: port must be specified")
		return
	}

	addr := ":" + strconv.Itoa(int(c.Port))
	c.client, err = rpc.Dial("tcp", addr)
	if err != nil {
		return
	}

	return
}

func (c *Client) Close() (err error) {
	if c.client != nil {
		err = c.client.Close()
		return
	}

	return
}

func (c *Client) Execute(streamConfig *core.StreamConfig) (msg string, err error) {
	var cmdResponse = new(core.CMDResponse)
	err = c.client.Call(core.HandlerName, streamConfig, cmdResponse)
	if err != nil {
		return
	}

	msg = cmdResponse.Message
	return

}
