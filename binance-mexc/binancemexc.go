package binancemexc

import (
	"sync/atomic"
	"time"

	binancesdk "github.com/adshao/go-binance/v2"
	"github.com/isther/arbitrage/binance"
	"github.com/isther/arbitrage/config"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

var (
	Paused         atomic.Bool
	klineRatioBase = decimal.NewFromInt(10000)
	pauseCh        = make(chan struct{})
	unPauseCh      = make(chan struct{})
)

func init() {
	Paused.Store(false)
	go Pause()
}

func Pause() {
	for {
		select {
		case <-pauseCh:
			Paused.Store(true)
		case <-unPauseCh:
			Paused.Store(false)
		}
	}
}

func StartCalculateKline() {
	var restartCh = make(chan struct{})
	for {
		_, _ = binance.StartKlineInfo(
			"BTCTUSD",
			"1m",
			func(event *binancesdk.WsKlineEvent) {
				high, _ := decimal.NewFromString(event.Kline.High)
				low, _ := decimal.NewFromString(event.Kline.Low)

				if high.Div(low).Sub(decimal.NewFromInt(1)).Mul(klineRatioBase).GreaterThanOrEqual(decimal.NewFromFloat(config.Config.KlineRatio)) {
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
}
