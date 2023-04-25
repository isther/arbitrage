package arbitrage

import (
	"log"
	"sync"

	"github.com/adshao/go-binance/v2"
)

type ArbitrageManager struct {
	lock sync.RWMutex

	symbolsMap map[string]struct{}
	symbols    []string

	handler    binance.WsBookTickerHandler
	errHandler binance.ErrHandler

	bookTickerMap map[string]*binance.WsBookTickerEvent

	restartCh chan struct{}
}

func NewArbitrageManager(symbols ...string) *ArbitrageManager {
	var b = ArbitrageManager{
		symbolsMap: make(map[string]struct{}),
		symbols:    []string{},

		bookTickerMap: make(map[string]*binance.WsBookTickerEvent),

		restartCh: make(chan struct{}),
	}

	b.SetHandler(func(event *binance.WsBookTickerEvent) {
		log.Printf("%+v\n", event)

		b.lock.Lock()
		defer b.lock.Unlock()

		b.bookTickerMap[event.Symbol] = event
	})

	b.SetErrHandler(func(err error) { panic(err) })

	b.addSymbol(symbols...)

	return &b
}

func (b *ArbitrageManager) StartWsBookTicker() {
	for {
		doneC, stopC := b.startWebsocket()
		log.Println("[BookTicker] Start binance websocket")

		select {
		case <-b.restartCh:
			b.updateSymbols()
			log.Println("[BookTicker] Restart websocket")
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

func (b *ArbitrageManager) GetBookTickerEvent(
	symbolA, symbolB, symbolC string) (
	*binance.WsBookTickerEvent, *binance.WsBookTickerEvent, *binance.WsBookTickerEvent) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	return b.bookTickerMap[symbolA], b.bookTickerMap[symbolB], b.bookTickerMap[symbolC]
}

func (b *ArbitrageManager) SetHandler(handler binance.WsBookTickerHandler) {
	b.handler = handler
}

func (b *ArbitrageManager) SetErrHandler(errHandler binance.ErrHandler) {
	b.errHandler = errHandler
}

func (b *ArbitrageManager) startWebsocket() (chan struct{}, chan struct{}) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	log.Println(b.symbols)
	doneC, stopC, err := binance.WsCombinedBookTickerServe(b.symbols, b.handler, b.errHandler)
	if err != nil {
		panic(err)
	}
	return doneC, stopC
}

func (b *ArbitrageManager) addSymbol(symbols ...string) *ArbitrageManager {
	func() {
		b.lock.Lock()
		defer b.lock.Unlock()

		for i := range symbols {
			b.symbolsMap[symbols[i]] = struct{}{}
		}
	}()

	b.updateSymbols()

	return b
}

func (b *ArbitrageManager) removeSymbol(symbols ...string) *ArbitrageManager {
	func() {
		b.lock.Lock()
		defer b.lock.Unlock()

		for i := range symbols {
			for j := range b.symbolsMap {
				if symbols[i] == j {
					delete(b.symbolsMap, j)
				}
			}
		}
	}()

	b.updateSymbols()

	return b
}

func (b *ArbitrageManager) updateSymbols() {
	b.lock.RLock()
	defer b.lock.RUnlock()

	b.symbols = make([]string, 0, len(b.symbolsMap))
	for k := range b.symbolsMap {
		b.symbols = append(b.symbols, k)
	}
}
