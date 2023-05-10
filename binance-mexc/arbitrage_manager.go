package binancemexc

import (
	"fmt"
	"log"
	"sync"

	binancesdk "github.com/adshao/go-binance/v2"
	"github.com/isther/arbitrage/binance"
	"github.com/isther/arbitrage/mexc"
)

type ArbitrageManager struct {
	lock sync.RWMutex

	symbolPairs SymbolPair

	// websocket server
	websocketApiServiceManager *binance.WebsocketServiceManager

	binanceHandler          binancesdk.WsBookTickerHandler
	binanceErrHandler       binancesdk.ErrHandler
	binanceSymbolEventCh    chan *binancesdk.WsBookTickerEvent
	stableCoinSymbolEventCh chan *binancesdk.WsBookTickerEvent

	mexcHandler       mexc.WsBookTickerHandler
	mexcErrHandler    mexc.ErrHandler
	mexcSymbolEventCh chan *mexc.WsBookTickerEvent

	restartCh chan struct{}
}

type SymbolPair struct {
	BinanceSymbol    string
	StableCoinSymbol string
	MexcSymbol       string
}

func NewArbitrageManager(symbolPairs SymbolPair) *ArbitrageManager {
	var b = ArbitrageManager{
		symbolPairs: symbolPairs, binanceSymbolEventCh: make(chan *binancesdk.WsBookTickerEvent),

		websocketApiServiceManager: binance.NewWebsocketServiceManager(),

		stableCoinSymbolEventCh: make(chan *binancesdk.WsBookTickerEvent),
		mexcSymbolEventCh:       make(chan *mexc.WsBookTickerEvent),
		restartCh:               make(chan struct{}),
	}

	b.SetBinanceHandler(func(event *binancesdk.WsBookTickerEvent) {
		switch event.Symbol {
		case b.symbolPairs.BinanceSymbol:
			b.binanceSymbolEventCh <- event
		case b.symbolPairs.StableCoinSymbol:
			b.stableCoinSymbolEventCh <- event
		}
	})

	b.SetBinanceErrHandler(func(err error) { panic(err) })

	b.SetMexcHandler(func(event *mexc.WsBookTickerEvent) {
		b.mexcSymbolEventCh <- event
	})

	b.SetMexcErrHandler(func(err error) { panic(err) })

	return &b
}

func (b *ArbitrageManager) Run() {
	go func() {
		restartCh := make(chan struct{})
		for {
			// b.wsRequestCh, b.wsResponseCh, doneC, stopC = b.websocketApiServiceManager.StartWsApi(
			_, _ = b.websocketApiServiceManager.StartWsApi(
				func(msg []byte) {
					wsApiEvent, method, err := b.websocketApiServiceManager.ParseWsApiEvent(msg)
					if err != nil {
						log.Println("[ERROR] Failed to parse wsApiEvent:", err)
						return
					}

					log.Println(fmt.Sprintf("[%s]: %+v", method, wsApiEvent))
				},
				func(err error) {
					restartCh <- struct{}{}
					// panic(err)
				},
			)
			<-restartCh
		}
	}()

	go b.startWsBookTicker()
}

func (b *ArbitrageManager) StartTask(task *Task) {
	task.run(b.binanceSymbolEventCh, b.stableCoinSymbolEventCh, b.mexcSymbolEventCh, b.websocketApiServiceManager.RequestCh)
}

func (b *ArbitrageManager) startWsBookTicker() {
	go func() {
		for {
			doneC, stopC := b.startBinanceBookTickerWebsocket()
			log.Println("[BookTicker] Start binance websocket")

			select {
			case <-b.restartCh:
				log.Println("[BookTicker] Restart websocket")
			case <-doneC:
				log.Println("[BookTicker] Done")
			}

			stopC <- struct{}{}
		}
	}()

	go func() {

		for {
			doneC, stopC := b.startMexcBookTickerWebsocket()
			log.Println("[BookTicker] Start mexc websocket")

			select {
			case <-b.restartCh:
				log.Println("[BookTicker] Restart mexc websocket")
			case <-doneC:
				log.Println("[BookTicker] Done")
			}

			stopC <- struct{}{}
		}
	}()
}

func (b *ArbitrageManager) Restart() *ArbitrageManager {
	b.restartCh <- struct{}{}

	return b
}

func (b *ArbitrageManager) SetBinanceHandler(handler binancesdk.WsBookTickerHandler) {
	b.binanceHandler = handler
}

func (b *ArbitrageManager) SetBinanceErrHandler(errHandler binancesdk.ErrHandler) {
	b.binanceErrHandler = errHandler
}

func (b *ArbitrageManager) startBinanceBookTickerWebsocket() (chan struct{}, chan struct{}) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	doneC, stopC, err := binancesdk.WsCombinedBookTickerServe(
		[]string{b.symbolPairs.BinanceSymbol, b.symbolPairs.StableCoinSymbol},
		b.binanceHandler,
		b.binanceErrHandler,
	)
	if err != nil {
		panic(err)
	}
	return doneC, stopC
}

func (b *ArbitrageManager) SetMexcHandler(handler mexc.WsBookTickerHandler) {
	b.mexcHandler = handler
}

func (b *ArbitrageManager) SetMexcErrHandler(errHandler mexc.ErrHandler) {
	b.mexcErrHandler = errHandler
}

func (b *ArbitrageManager) startMexcBookTickerWebsocket() (chan struct{}, chan struct{}) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	doneC, stopC, err := mexc.WsBookTickerServe(
		b.symbolPairs.MexcSymbol,
		b.mexcHandler,
		b.mexcErrHandler,
	)
	if err != nil {
		panic(err)
	}
	return doneC, stopC
}
