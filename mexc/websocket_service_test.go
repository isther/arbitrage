package mexc

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/isther/arbitrage/config"
	"github.com/stretchr/testify/assert"
)

var (
	wsHandler = func(message []byte) {
		fmt.Println(string(message))
	}

	errHandler = func(err error) { panic(err) }
)

func TestCreateListenKey(t *testing.T) {
	config.Load("../config.yaml")
	// t.Log(CreateListenKey())

	key, err := NewClient(config.Config.MexcApiKey, config.Config.MexcSecretKey).NewListenKeyService().Do(context.Background())
	if err != nil {
		panic(err)
	}
	t.Log(key)
}

func TestWsDepthServe(t *testing.T) {
	assert := assert.New(t)
	doneC, stopC, err := WsDepthServe("BTCUSDT", wsHandler, errHandler)
	assert.Nil(err)

	go func() {
		time.Sleep(10 * time.Second)
		stopC <- struct{}{}
	}()

	<-doneC
}

func TestWsPartialDepthServe(t *testing.T) {
	assert := assert.New(t)
	doneC, stopC, err := WsPartialDepthServe("BTCUSDT", "5", wsHandler, errHandler)
	assert.Nil(err)

	go func() {
		time.Sleep(10 * time.Second)
		stopC <- struct{}{}
	}()

	<-doneC
}

func TestWsBookTickerServe(t *testing.T) {
	assert := assert.New(t)
	doneC, stopC, err := WsBookTickerServe("BTCUSDT", func(event *WsBookTickerEvent) { fmt.Printf("%+v\n", event) }, errHandler)
	assert.Nil(err)

	go func() {
		time.Sleep(10 * time.Second)
		stopC <- struct{}{}
	}()

	<-doneC
}

func TestWsAccountInfoServe(t *testing.T) {
	assert := assert.New(t)
	config.Load("../config.yaml")
	doneC, stopC, err := WsAccountInfoServe(func(event *WsPrivateAccountEvent) { fmt.Printf("%+v\n", event) }, errHandler)
	assert.Nil(err)

	go func() {
		time.Sleep(10 * time.Second)
		stopC <- struct{}{}
	}()

	<-doneC
}

func TestWsDealsInfoServe(t *testing.T) {
	assert := assert.New(t)
	config.Load("../config.yaml")
	doneC, stopC, err := WsDealsInfoServe(CreateListenKey(), func(event *WsPrivateDealsEvent) { fmt.Printf("%+v\n", event) }, errHandler)
	assert.Nil(err)

	go func() {
		time.Sleep(10 * time.Second)
		stopC <- struct{}{}
	}()

	<-doneC
}
