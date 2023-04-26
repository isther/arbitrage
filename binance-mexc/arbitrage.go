package binancemexc

import (
	"log"

	"github.com/isther/arbitrage/websocket-api/binance"
)

var (
	TestTrade                = false
	ArbitrageManagerInstance *ArbitrageManager
	// L                        sync.RWMutex

	WebsocketApiServiceManager = binance.NewWebsocketServiceManager()

	requestCh  = make(chan *binance.WsApiRequest)
	responseCh = make(chan *binance.WsApiEvent)
	doneC      chan struct{}
	stopC      chan struct{}
)

func init() {

	go func() {
		var (
			restartCh = make(chan struct{})
		)
		for {
			requestCh, responseCh, doneC, stopC = WebsocketApiServiceManager.StartWsApi(
				func(msg []byte) {
					wsApiEvent, method, err := WebsocketApiServiceManager.ParseWsApiEvent(msg)
					if err != nil {
						log.Println("[ERROR] Failed to parse wsApiEvent:", err)
						return
					}

					switch method {
					case binance.Ping:
					case binance.ServerTime:
					case binance.ExchangeInfo:
					case binance.AccountStatus:
					case binance.OrderTrade:
					}

					responseCh <- wsApiEvent
				},
				func(err error) {
					restartCh <- struct{}{}
					// panic(err)
				},
			)
			<-restartCh
		}
	}()

	go func() {
		for {
			response := <-responseCh
			log.Println(response)
		}
	}()
}
