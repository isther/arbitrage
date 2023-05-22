package dingding

import (
	"context"
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
	for {
		func() {
			var (
				msg         = ""
				ctx, cancel = context.WithTimeout(context.Background(), 3000*time.Millisecond)
			)
			defer cancel()

			for {
				select {
				case m := <-d.MsgCh:
					msg += m
				case <-ctx.Done():
					if msg != "" {
						d.client.Send(dingtalk.NewTextMessage().SetContent(msg))
						return
					}
				}
			}
		}()
		time.Sleep(10 * time.Millisecond)
	}
}
