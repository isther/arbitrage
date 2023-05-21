package dingding

import (
	"testing"
	"time"

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

	go func() {
		for {
			time.Sleep(100 * time.Millisecond)
			dingBot.MsgCh <- "Hello\n"
		}
	}()

	time.Sleep(10 * time.Second)
}
