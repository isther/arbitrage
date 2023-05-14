package binancemexc

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	binancesdk "github.com/adshao/go-binance/v2"
	"github.com/isther/arbitrage/binance"
	"github.com/isther/arbitrage/mexc"
	"github.com/sirupsen/logrus"
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
		symbolPairs:          symbolPairs,
		binanceSymbolEventCh: make(chan *binancesdk.WsBookTickerEvent),

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

func (b *ArbitrageManager) Run(
	openBinanceOrderIdCh chan string,
	closeBinanceOrderIdCh chan string,
) {
	var (
		started              atomic.Bool
		startBinanceWsDoneCh = make(chan struct{})
		startMexcWsDoneCh    = make(chan struct{})
	)

	started.Store(false)
	go func() {
		restartCh := make(chan struct{})
		for {
			_, _ = b.websocketApiServiceManager.StartWsApi(
				func(msg []byte) {
					wsApiEvent, method, err := b.websocketApiServiceManager.ParseWsApiEvent(msg)
					if err != nil {
						logrus.Error("Failed to parse wsApiEvent:", err)
						return
					}

					switch method {
					case binance.OrderTrade:
						if strings.HasPrefix(wsApiEvent.OrderTradeResponse.ClientOrderID, "C") ||
							strings.HasPrefix(wsApiEvent.OrderTradeResponse.ClientOrderID, "FC") {
							logrus.Infof("[平仓] Price: %s Qty: %s",
								wsApiEvent.OrderTradeResponse.Price,
								wsApiEvent.OrderTradeResponse.OrigQuantity,
							)
						} else if strings.HasPrefix(wsApiEvent.OrderTradeResponse.ClientOrderID, "O") {
							logrus.Infof("[开仓] Price: %s Qty: %s",
								wsApiEvent.OrderTradeResponse.Price,
								wsApiEvent.OrderTradeResponse.OrigQuantity,
							)
						}
					default:
						logrus.Info(fmt.Sprintf("[%s]: %+v", method, wsApiEvent))
					}
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
			doneC, stopC, err := b.startBinanceBookTickerWebsocket()
			if err != nil {
				continue
			}
			logrus.Info("[BookTicker] Start binance websocket")

			if !started.Load() {
				startBinanceWsDoneCh <- struct{}{}
			}

			select {
			case <-b.restartCh:
				logrus.Info("[BookTicker] Restart websocket")
			case <-doneC:
				logrus.Info("[BookTicker] Done")
			}

			stopC <- struct{}{}
		}
	}()

	go func() {
		for {
			doneC, stopC, err := b.startMexcBookTickerWebsocket()
			if err != nil {
				continue
			}
			logrus.Info("[BookTicker] Start mexc websocket")

			if !started.Load() {
				startMexcWsDoneCh <- struct{}{}
			}

			select {
			case <-b.restartCh:
				logrus.Warnf("[BookTicker] Restart mexc websocket")
			case <-doneC:
				logrus.Info("[BookTicker] Done")
			}

			stopC <- struct{}{}
		}
	}()

	<-startBinanceWsDoneCh
	<-startMexcWsDoneCh
	started.Store(true)
}

func (b *ArbitrageManager) StartTask(task *Task, openMexcOrderIdCh, closeMexcOrderIdCh chan string) {
	task.run(b.websocketApiServiceManager.RequestCh,
		openMexcOrderIdCh, closeMexcOrderIdCh,
		b.binanceSymbolEventCh,
		b.stableCoinSymbolEventCh,
		b.mexcSymbolEventCh,
	)
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

func (b *ArbitrageManager) startBinanceBookTickerWebsocket() (chan struct{}, chan struct{}, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	return binancesdk.WsCombinedBookTickerServe(
		[]string{b.symbolPairs.BinanceSymbol, b.symbolPairs.StableCoinSymbol},
		b.binanceHandler,
		b.binanceErrHandler,
	)
}

func (b *ArbitrageManager) SetMexcHandler(handler mexc.WsBookTickerHandler) {
	b.mexcHandler = handler
}

func (b *ArbitrageManager) SetMexcErrHandler(errHandler mexc.ErrHandler) {
	b.mexcErrHandler = errHandler
}

func (b *ArbitrageManager) startMexcBookTickerWebsocket() (chan struct{}, chan struct{}, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	return mexc.WsBookTickerServe(
		b.symbolPairs.MexcSymbol,
		b.mexcHandler,
		b.mexcErrHandler,
	)
}
