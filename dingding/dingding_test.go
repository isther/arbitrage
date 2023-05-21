package dingding

import (
	"testing"
	"time"

	"github.com/CatchZeng/dingtalk/pkg/dingtalk"
	"github.com/isther/arbitrage/config"
)

func TestDingDingBot(t *testing.T) {
	config.Load("../config.yaml")

	dingBot := NewDingDingBot(
		config.Config.LogDingDingConfig.AccessToken,
		config.Config.LogDingDingConfig.Secrect,
		100,
	)
	go dingBot.Start()

	dingBot.MsgCh <- dingtalk.NewMarkdownMessage().SetMarkdown("Hello", "Test")

	time.Sleep(1 * time.Second)
}
