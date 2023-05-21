package dingding

import (
	"sync"
	"time"

	"github.com/CatchZeng/dingtalk/pkg/dingtalk"
)

type DingDingBot struct {
	client *dingtalk.Client
	// MsgCh  chan dingtalk.Message
	MsgCh chan string
}

func NewDingDingBot(accessToken, secrect string, chLen int) *DingDingBot {
	return &DingDingBot{
		client: dingtalk.NewClient(accessToken, secrect),
		// MsgCh:  make(chan dingtalk.Message, chLen),
		MsgCh: make(chan string, chLen),
	}
}

func (d *DingDingBot) Start() {
	var (
		L   sync.Mutex
		msg = ""
	)

	go func() {
		time.Sleep(1000 * time.Millisecond)
		for {
			func() {
				L.Lock()
				defer L.Unlock()
				if msg != "" {
					d.client.Send(dingtalk.NewActionCardMessage().SetIndependentJump("Msg", msg, nil, "", ""))
					msg = ""
				}
			}()
			time.Sleep(3001 * time.Millisecond)
		}
	}()

	for {
		func() {
			L.Lock()
			defer L.Unlock()

			msg += <-d.MsgCh
		}()
	}
}
