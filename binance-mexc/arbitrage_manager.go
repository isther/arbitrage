package binancemexc

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	binancesdk "github.com/adshao/go-binance/v2"
	"github.com/isther/arbitrage/binance"
	"github.com/isther/arbitrage/config"
	"github.com/isther/arbitrage/mexc"
	"github.com/shopspring/decimal"
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

func (b *ArbitrageManager) Start() {
	var (
		started atomic.Bool
		wg      sync.WaitGroup
	)

	started.Store(false)

	go func() {
		wg.Add(1)
		for {
			doneC, stopC, err := b.startBinanceBookTickerWebsocket()
			if err != nil {
				continue
			}
			logrus.Debug("[BookTicker] Start binance websocket")

			if !started.Load() {
				wg.Done()
			}

			select {
			case <-b.restartCh:
				logrus.Debug("[BookTicker] Restart websocket")
			case <-doneC:
				logrus.Debug("[BookTicker] Done")
			}

			stopC <- struct{}{}
		}
	}()

	go func() {
		wg.Add(1)
		for {
			doneC, stopC, err := b.startMexcBookTickerWebsocket()
			if err != nil {
				continue
			}
			logrus.Debug("[BookTicker] Start mexc websocket")

			if !started.Load() {
				wg.Done()
			}

			select {
			case <-b.restartCh:
				logrus.Debug("[BookTicker] Restart mexc websocket")
			case <-doneC:
				logrus.Debug("[BookTicker] Done")
			}

			stopC <- struct{}{}
		}
	}()

	go func() {
		wg.Add(1)
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
						} else if strings.HasPrefix(wsApiEvent.OrderTradeResponse.ClientOrderID, "O") {
						}
					case binance.ServerTime:
						if decimal.NewFromInt(time.Now().UnixMilli() - wsApiEvent.ServerTime.ServerTime).
							GreaterThanOrEqual(decimal.NewFromInt(config.Config.ClientTimeOut)) {
							go func() {
								if !Paused.Load() {
									pauseCh <- struct{}{}
									logrus.Warn("币安超时，已暂停")
									time.Sleep(time.Duration(config.Config.ClientTimeOutPauseDuration) * time.Millisecond)
									logrus.Warn("币安超时暂停结束")
									unPauseCh <- struct{}{}
								}
							}()
						}
					default:
						logrus.Debug(fmt.Sprintf("[%s]: %+v", method, wsApiEvent))
					}
				},
				func(err error) {
					restartCh <- struct{}{}
					// panic(err)
				},
			)
			if !started.Load() {
				wg.Done()
			}

			<-restartCh
		}
	}()

	wg.Wait()
	started.Store(true)

	// Send request to get server time
	go func() {
		for {
			b.websocketApiServiceManager.RequestCh <- binance.NewServerTime()
			time.Sleep(1 * time.Second)
		}
	}()

	go func() {
		var restartCh = make(chan struct{})
		for {
			_, _ = binance.StartKlineInfo(
				"BTCTUSD",
				"1m",
				func(event *binancesdk.WsKlineEvent) {
					high, _ := decimal.NewFromString(event.Kline.High)
					low, _ := decimal.NewFromString(event.Kline.Low)

					if high.Div(low).Sub(decimal.NewFromInt(1)).Mul(klineRatioBase).
						GreaterThanOrEqual(decimal.NewFromFloat(config.Config.KlineRatio)) {
						go func() {
							if !Paused.Load() {
								pauseCh <- struct{}{}
								logrus.Warn("BTC振幅过高，已暂停")
								time.Sleep(time.Duration(config.Config.KlinePauseDuration) * time.Millisecond)
								logrus.Warn("BTC振幅过高暂停结束")
								unPauseCh <- struct{}{}
							}
						}()
					}
				},
				func(err error) {
					if err != nil {
						logrus.WithFields(logrus.Fields{"server": "kline"}).Error(err)
						restartCh <- struct{}{}
					}
				},
			)

			<-restartCh
		}
	}()

	go func() {
		for {
			serverTime := mexc.ServerTime()
			if decimal.NewFromInt(time.Now().UnixMilli() - serverTime).
				GreaterThanOrEqual(decimal.NewFromInt(config.Config.ClientTimeOut)) {
				go func() {
					if !Paused.Load() {
						pauseCh <- struct{}{}
						logrus.Warn("抹茶超时，已暂停")
						time.Sleep(time.Duration(config.Config.ClientTimeOutPauseDuration) * time.Millisecond)
						logrus.Warn("抹茶超时暂停结束")
						unPauseCh <- struct{}{}
					}
				}()
			}
			time.Sleep(1 * time.Second)
		}
	}()
}

func (b *ArbitrageManager) StartTask(task *Task, OrderIDsCh chan OrderIds) {
	task.run(b.websocketApiServiceManager.RequestCh,
		b.binanceSymbolEventCh,
		b.stableCoinSymbolEventCh,
		b.mexcSymbolEventCh,
		OrderIDsCh,
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
