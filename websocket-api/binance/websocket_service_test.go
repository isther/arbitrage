package binance

import (
	"log"
	"sync"
	"testing"

	"github.com/isther/arbitrage/config"
)

var (
	WebsocketApiServiceManager = NewWebsocketServiceManager()

	requestCh  = make(chan *WsApiRequest)
	responseCh = make(chan *WsApiEvent)
	doneC      chan struct{}
	stopC      chan struct{}
)

func init() {
	config.Load("config.yaml")

	requestCh, responseCh, doneC, stopC = WebsocketApiServiceManager.StartWsApi(
		func(msg []byte) {
			wsApiEvent, method, err := WebsocketApiServiceManager.ParseWsApiEvent(msg)
			if err != nil {
				log.Println("[ERROR] Failed to parse wsApiEvent:", err)
				return
			}

			switch method {
			case Ping:
			case ServerTime:
			case ExchangeInfo:
			case AccountStatus:
			case OrderTrade:
			}

			responseCh <- wsApiEvent
		},
		func(err error) {
			panic(err)
		},
	)
}

func TestWsApiMethod(t *testing.T) {
	var (
		apiKey    = config.Config.BinanceApiKey
		secretKey = config.Config.BinanceSecretKey
		w         sync.WaitGroup
	)

	go func() {
		for {
			response := <-responseCh
			log.Println(response)
			w.Done()
		}
	}()

	// Ping
	ping := NewPing()
	requestCh <- ping
	w.Add(1)

	// ServerTime
	serverTime := NewServerTime()
	requestCh <- serverTime
	w.Add(1)

	// ExchangeInfo
	exchangeInfo := NewSpotExchangeInfo()
	requestCh <- exchangeInfo
	w.Add(1)

	// AccountStatus
	accountStatus := NewAccountStatus(apiKey, secretKey)
	requestCh <- accountStatus
	w.Add(1)

	// OrderTrade
	params := NewOrderTradeParmes(apiKey).
		Symbol("BTCUSDT").Side(SideTypeSell).OrderType(OrderTypeLimit).
		NewOrderRespType(NewOrderRespTypeACK).TimeInForce(TimeInForceTypeGTC).
		Price("28050.00").Quantity("0.01000000").
		Signature(secretKey)

	// orderTrade := NewOrderTrade(params)
	// requestCh <- orderTrade
	// w.Add(1)

	// Test
	orderTradeTest := NewOrderTradeTest(params)
	requestCh <- orderTradeTest
	w.Add(1)

	w.Wait()
}
