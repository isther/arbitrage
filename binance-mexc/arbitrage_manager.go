package binancemexc

import (
	"context"
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

var (
	binanceWsServiceRestartCh = make(chan struct{})
)

type ArbitrageManager struct {
	lock sync.RWMutex

	symbolPairs SymbolPair

	// websocket server
	// websocketApiServiceManager *binance.WebsocketServiceManager

	binanceSymbolEventCh    chan *binancesdk.WsBookTickerEvent
	stableCoinSymbolEventCh chan *binancesdk.WsBookTickerEvent

	// mexcSymbolEventCh chan *mexc.WsBookTickerEvent
	mexcSymbolEventCh chan *binancesdk.WsBookTickerEvent
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

		// websocketApiServiceManager: binance.NewWebsocketServiceManager(),

		stableCoinSymbolEventCh: make(chan *binancesdk.WsBookTickerEvent),
		// mexcSymbolEventCh:       make(chan *mexc.WsBookTickerEvent),
		mexcSymbolEventCh: make(chan *binancesdk.WsBookTickerEvent),
	}

	return &b
}

func (b *ArbitrageManager) Start() {
	var (
		started atomic.Bool
		wg      sync.WaitGroup
	)

	// go func() {
	// 	for {
	// 		b.websocketApiServiceManager.Send(<-tradeRequestCh)
	// 	}
	// }()

	started.Store(false)

	wg.Add(1)
	go func() {
		restartCh := make(chan struct{})
		for {
			doneC, _ := b.startBinanceBookTickerWebsocket(
				func(event *binancesdk.WsBookTickerEvent) {
					switch event.Symbol {
					case b.symbolPairs.BinanceSymbol:
						b.binanceSymbolEventCh <- event
					case b.symbolPairs.StableCoinSymbol:
						b.stableCoinSymbolEventCh <- event
					case b.symbolPairs.MexcSymbol:
						b.mexcSymbolEventCh <- event
					}
				},
				func(err error) {
					logrus.Error("BinanceBookTicker: ", err)
					restartCh <- struct{}{}
					// panic(err)
				},
			)
			logrus.Debug("[BookTicker] Start binance websocket")

			if !started.Load() {
				wg.Done()
			}

			select {
			case <-restartCh:
				logrus.Debug("[BookTicker] Restart websocket")
			case <-doneC:
				logrus.Debug("[BookTicker] Done")
			}
		}
	}()

	// wg.Add(1)
	// go func() {
	// 	restartCh := make(chan struct{})
	// 	for {
	// 		doneC, _ := b.startMexcBookTickerWebsocket(
	// 			func(event *mexc.WsBookTickerEvent) {
	// 				b.mexcSymbolEventCh <- event
	// 			},
	// 			func(err error) {
	// 				logrus.Error("MexcBookTicker: ", err)
	// 				restartCh <- struct{}{}
	// 				// panic(err)
	// 			},
	// 		)
	// 		logrus.Debug("[BookTicker] Start mexc websocket")
	//
	// 		if !started.Load() {
	// 			wg.Done()
	// 		}
	//
	// 		select {
	// 		case <-restartCh:
	// 			logrus.Debug("[BookTicker] Restart mexc bookticker websocket")
	// 		case <-doneC:
	// 			logrus.Debug("[BookTicker] Done")
	// 		}
	// 	}
	// }()

	// wg.Add(1)
	// go func() {
	// 	// Send request to get server time
	// 	go func() {
	// 		for {
	// 			b.websocketApiServiceManager.Send(binance.NewServerTime())
	// 			time.Sleep(1 * time.Second)
	// 		}
	// 	}()
	//
	// 	for {
	// 		_, _ = b.websocketApiServiceManager.StartWsApi(
	// 			func(msg []byte) {
	// 				wsApiEvent, method, err := b.websocketApiServiceManager.ParseWsApiEvent(msg)
	// 				if err != nil {
	// 					logrus.Error("Failed to parse wsApiEvent:", err)
	// 					return
	// 				}
	//
	// 				switch method {
	// 				case binance.OrderTrade:
	// 					if strings.HasPrefix(wsApiEvent.OrderTradeResponse.ClientOrderID, "C") ||
	// 						strings.HasPrefix(wsApiEvent.OrderTradeResponse.ClientOrderID, "FC") {
	// 					} else if strings.HasPrefix(wsApiEvent.OrderTradeResponse.ClientOrderID, "O") {
	// 						// logrus.Info(wsApiEvent.RateLimits)
	// 					}
	// 					logrus.Info("tradeInfo", wsApiEvent.OrderTradeResponse)
	// 				case binance.ServerTime:
	// 					if decimal.NewFromInt(time.Now().UnixMilli() - wsApiEvent.ServerTime.ServerTime).
	// 						GreaterThanOrEqual(decimal.NewFromInt(config.Config.ClientTimeOut)) {
	// 						if !Paused.Load() {
	// 							pauseCh <- struct{}{}
	// 							logrus.Warn("币安超时，已暂停")
	// 							time.Sleep(time.Duration(config.Config.ClientTimeOutPauseDuration) * time.Millisecond)
	// 							logrus.Warn("币安超时暂停结束")
	// 							unPauseCh <- struct{}{}
	// 						}
	// 					}
	// 					// default:
	// 					// logrus.Warn(fmt.Sprintf("[%s]: %+v", method, wsApiEvent))
	// 				}
	// 			},
	// 			func(err error) {
	// 				logrus.Error("BinanceWsApiServiceManager: ", err)
	// 				binanceWsServiceRestartCh <- struct{}{}
	// 				// panic(err)
	// 			},
	// 		)
	// 		if !started.Load() {
	// 			wg.Done()
	// 		}
	//
	// 		<-binanceWsServiceRestartCh
	// 		logrus.Debug("BinanceWsApiServiceManager: Restart")
	// 	}
	// }()

	wg.Wait()
	started.Store(true)

	go b.startCheckBinanceKline()
	// go b.startCheckMexcServerTime()
	go b.startCheckBinanceServerTime()
}

func (b *ArbitrageManager) StartTask(task *Task, OrderIDsCh chan OrderIds) {
	task.run(
		b.binanceSymbolEventCh,
		b.stableCoinSymbolEventCh,
		b.mexcSymbolEventCh,
		OrderIDsCh,
	)
}

func (b *ArbitrageManager) startBinanceBookTickerWebsocket(
	handler binancesdk.WsBookTickerHandler, errHandler binancesdk.ErrHandler,
) (chan struct{}, chan struct{}) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	var (
		err   error
		doneC chan struct{}
		stopC chan struct{}
	)
	for {
		doneC, stopC, err = binancesdk.WsCombinedBookTickerServe(
			[]string{b.symbolPairs.BinanceSymbol, b.symbolPairs.StableCoinSymbol, b.symbolPairs.MexcSymbol},
			handler,
			errHandler,
		)

		if err == nil {
			break
		}
		logrus.Error(err)
	}
	logrus.Debug("Connect to mexc bookticker info websocket server successfully.")

	return doneC, stopC
}

func (b *ArbitrageManager) startMexcBookTickerWebsocket(
	handler mexc.WsBookTickerHandler, errHandler mexc.ErrHandler,
) (chan struct{}, chan struct{}) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	var (
		err   error
		doneC chan struct{}
		stopC chan struct{}
	)
	for {
		doneC, stopC, err = mexc.WsBookTickerServe(
			b.symbolPairs.MexcSymbol,
			handler,
			errHandler,
		)
		if err == nil {
			break
		}
		logrus.Error(err)
	}
	logrus.Debug("Connect to mexc bookticker info websocket server successfully.")

	return doneC, stopC
}

func (b *ArbitrageManager) startCheckBinanceKline() {
	var restartCh = make(chan struct{})
	for {
		_, _ = binance.StartKlineInfo(
			"BTCTUSD",
			"1m",
			func(event *binancesdk.WsKlineEvent) {
				high, _ := decimal.NewFromString(event.Kline.High)
				low, _ := decimal.NewFromString(event.Kline.Low)

				ratio := high.Div(low).Sub(decimal.NewFromInt(1)).Mul(klineRatioBase)

				if ratio.GreaterThanOrEqual(decimal.NewFromFloat(config.Config.MaxKlineRatio)) {
					if !Paused.Load() {
						pauseCh <- struct{}{}
						logrus.Warn("BTC振幅过高，已暂停")
						time.Sleep(time.Duration(config.Config.KlinePauseDuration) * time.Millisecond)
						logrus.Warn("BTC振幅过高暂停结束")
						unPauseCh <- struct{}{}
					}
				} else if ratio.LessThanOrEqual(decimal.NewFromFloat(config.Config.MinKlineRatio)) {
					if !Paused.Load() {
						pauseCh <- struct{}{}
						logrus.Warn("BTC振幅过低，已暂停")
						time.Sleep(time.Duration(config.Config.KlinePauseDuration) * time.Millisecond)
						logrus.Warn("BTC振幅过低暂停结束")
						unPauseCh <- struct{}{}
					}
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
}

func (b *ArbitrageManager) startCheckMexcServerTime() {
	for {
		serverTime, err := newMexcClient().NewServerTimeService().Do(context.Background())
		if err != nil {
			mexc.Logger.Error("Failed to get server time:", err)
			continue
		}
		if decimal.NewFromInt(time.Now().UnixMilli() - serverTime).
			GreaterThanOrEqual(decimal.NewFromInt(config.Config.ClientTimeOut)) {
			if !Paused.Load() {
				pauseCh <- struct{}{}
				logrus.Warn("抹茶超时，已暂停")
				time.Sleep(time.Duration(config.Config.ClientTimeOutPauseDuration) * time.Millisecond)
				logrus.Warn("抹茶超时暂停结束")
				unPauseCh <- struct{}{}
			}
		}
		time.Sleep(1 * time.Second)
	}
}

func (b *ArbitrageManager) startCheckBinanceServerTime() {
	for {
		serverTime, err := newBinanceClient().NewServerTimeService().Do(context.Background())
		if err != nil {
			binance.Logger.Error("Failed to get server time:", err)
			continue
		}

		if decimal.NewFromInt(time.Now().UnixMilli() - serverTime).
			GreaterThanOrEqual(decimal.NewFromInt(config.Config.ClientTimeOut)) {
			if !Paused.Load() {
				pauseCh <- struct{}{}
				logrus.Warn("币安超时，已暂停")
				time.Sleep(time.Duration(config.Config.ClientTimeOutPauseDuration) * time.Millisecond)
				logrus.Warn("币安超时暂停结束")
				unPauseCh <- struct{}{}
			}
		}
		time.Sleep(1 * time.Second)
	}
}
