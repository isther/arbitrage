package binancemexc

import (
	"log"
	"sync"

	"github.com/adshao/go-binance/v2"
	"github.com/isther/arbitrage/mexc"
)

type ArbitrageManager struct {
	lock sync.RWMutex

	symbolPairs SymbolPair

	binanceHandler          binance.WsBookTickerHandler
	binanceErrHandler       binance.ErrHandler
	binanceSymbolEventCh    chan *binance.WsBookTickerEvent
	stableCoinSymbolEventCh chan *binance.WsBookTickerEvent

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
		symbolPairs: symbolPairs, binanceSymbolEventCh: make(chan *binance.WsBookTickerEvent),
		stableCoinSymbolEventCh: make(chan *binance.WsBookTickerEvent),
		mexcSymbolEventCh:       make(chan *mexc.WsBookTickerEvent),
		restartCh:               make(chan struct{}),
	}

	b.SetBinanceHandler(func(event *binance.WsBookTickerEvent) {
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

func (b *ArbitrageManager) StartWsBookTicker() {
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
}

func (b *ArbitrageManager) Restart() *ArbitrageManager {
	b.restartCh <- struct{}{}

	return b
}

func (b *ArbitrageManager) SetBinanceHandler(handler binance.WsBookTickerHandler) {
	b.binanceHandler = handler
}

func (b *ArbitrageManager) SetBinanceErrHandler(errHandler binance.ErrHandler) {
	b.binanceErrHandler = errHandler
}

func (b *ArbitrageManager) startBinanceBookTickerWebsocket() (chan struct{}, chan struct{}) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	doneC, stopC, err := binance.WsCombinedBookTickerServe(
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
