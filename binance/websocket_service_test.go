package binance

import (
	"fmt"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/isther/arbitrage/config"
)

func TestWsApiMethod(t *testing.T) {
	config.Load("../config.yaml")

	var (
		apiKey    = config.Config.BinanceApiKey
		secretKey = config.Config.BinanceSecretKey
		wg        sync.WaitGroup
		// doneC                      chan struct{}
		// stopC                      chan struct{}
		WebsocketApiServiceManager = NewWebsocketServiceManager()
	)

	// requestCh, responseCh, doneC, stopC = WebsocketApiServiceManager.StartWsApi(
	_, _ = WebsocketApiServiceManager.StartWsApi(
		func(msg []byte) {
			wsApiEvent, method, err := WebsocketApiServiceManager.ParseWsApiEvent(msg)
			if err != nil {
				log.Println("[ERROR] Failed to parse wsApiEvent:", err)
				return
			}

			log.Println(fmt.Sprintf("[%s]: %+v", method, wsApiEvent))
			wg.Done()
		},
		func(err error) {
			panic(err)
		},
	)

	// Ping
	WebsocketApiServiceManager.Send(NewPing())
	wg.Add(1)

	// ServerTime
	WebsocketApiServiceManager.Send(NewServerTime())
	wg.Add(1)

	// ExchangeInfo
	WebsocketApiServiceManager.Send(NewSpotExchangeInfo())
	wg.Add(1)

	// AccountStatus
	WebsocketApiServiceManager.Send(NewAccountStatus(apiKey, secretKey))
	wg.Add(1)

	// OrderTrade
	params := NewOrderTradeParmes(apiKey).
		Symbol("BTCUSDT").Side(SideTypeBuy).OrderType(OrderTypeLimit).
		NewOrderRespType(NewOrderRespTypeACK).TimeInForce(TimeInForceTypeGTC).
		Price("30000.000000").Quantity("0.0005101").
		Signature(secretKey)

	// WebsocketApiServiceManager.Send(NewOrderTrade(params))
	// w.Add(1)

	// Test
	WebsocketApiServiceManager.Send(NewOrderTradeTest(params))
	wg.Add(1)

	fmt.Println("test account...........")
	// AccountStatus
	// WebsocketApiServiceManager.Send(WsUserData(apiKey, secretKey))
	WsUserData(apiKey, secretKey)
	// wg.Add(1)

	wg.Wait()
}

func TestWsApiKeepLive(t *testing.T) {
	config.Load("../config.yaml")

	var (
		// apiKey    = config.Config.BinanceApiKey
		// secretKey = config.Config.BinanceSecretKey
		// doneC                      chan struct{}
		// stopC                      chan struct{}
		WebsocketApiServiceManager = NewWebsocketServiceManager()

		reqCnt   = 0
		replyCnt = 0
	)

	_, _ = WebsocketApiServiceManager.StartWsApi(
		func(msg []byte) {
			wsApiEvent, method, err := WebsocketApiServiceManager.ParseWsApiEvent(msg)
			// _, _, err := WebsocketApiServiceManager.ParseWsApiEvent(msg)
			if err != nil {
				log.Println("[ERROR] Failed to parse wsApiEvent:", err)
				return
			}

			log.Println(fmt.Sprintf("[%s]: %+v", method, wsApiEvent))
			replyCnt++
			t.Log(reqCnt, replyCnt)
		},
		func(err error) {
			panic(err)
		},
	)
	t.Log("start")

	for {
		time.Sleep(1 * time.Second)
		WebsocketApiServiceManager.Send(NewServerTime())
		reqCnt++
	}
}
