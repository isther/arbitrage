package dingding

import (
	"fmt"

	"github.com/CatchZeng/dingtalk/pkg/dingtalk"
)

type DingDingBot struct {
	client *dingtalk.Client
	MsgCh  chan dingtalk.Message
}

func NewDingDingBot(accessToken, secrect string, chLen int) *DingDingBot {
	return &DingDingBot{
		client: dingtalk.NewClient(accessToken, secrect),
		MsgCh:  make(chan dingtalk.Message, chLen),
	}
}

func (d *DingDingBot) Start() {
	for {
		msg := <-d.MsgCh
		re, res, err := d.client.Send(msg)
		fmt.Println(re, " ", res, " ", err)
	}
}
