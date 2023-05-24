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

	binanceSymbolEventCh    chan *binancesdk.WsBookTickerEvent
	stableCoinSymbolEventCh chan *binancesdk.WsBookTickerEvent

	mexcSymbolEventCh chan *mexc.WsBookTickerEvent
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
	}

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
		restartCh := make(chan struct{})
		for {
			doneC, _ := b.startBinanceBookTickerWebsocket(
				func(event *binancesdk.WsBookTickerEvent) {
					switch event.Symbol {
					case b.symbolPairs.BinanceSymbol:
						b.binanceSymbolEventCh <- event
					case b.symbolPairs.StableCoinSymbol:
						b.stableCoinSymbolEventCh <- event
					}
				},
				func(err error) {
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

	go func() {
		wg.Add(1)
		restartCh := make(chan struct{})
		for {
			doneC, _ := b.startMexcBookTickerWebsocket(
				func(event *mexc.WsBookTickerEvent) {
					b.mexcSymbolEventCh <- event
				},
				func(err error) {
					restartCh <- struct{}{}
					// panic(err)
				},
			)
			logrus.Debug("[BookTicker] Start mexc websocket")

			if !started.Load() {
				wg.Done()
			}

			select {
			case <-restartCh:
				logrus.Debug("[BookTicker] Restart mexc bookticker websocket")
			case <-doneC:
				logrus.Debug("[BookTicker] Done")
			}
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
							if !Paused.Load() {
								pauseCh <- struct{}{}
								logrus.Warn("币安超时，已暂停")
								time.Sleep(time.Duration(config.Config.ClientTimeOutPauseDuration) * time.Millisecond)
								logrus.Warn("币安超时暂停结束")
								unPauseCh <- struct{}{}
							}
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
						if !Paused.Load() {
							pauseCh <- struct{}{}
							logrus.Warn("BTC振幅过高，已暂停")
							time.Sleep(time.Duration(config.Config.KlinePauseDuration) * time.Millisecond)
							logrus.Warn("BTC振幅过高暂停结束")
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
	}()

	go func() {
		for {
			serverTime := mexc.ServerTime()
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
			[]string{b.symbolPairs.BinanceSymbol, b.symbolPairs.StableCoinSymbol},
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
