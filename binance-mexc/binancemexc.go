package binancemexc

import (
	"sync/atomic"

	"github.com/shopspring/decimal"
)

const (
	qty = "0.000379"
)

var (
	Paused         atomic.Bool
	pauseCh        = make(chan struct{})
	unPauseCh      = make(chan struct{})
	klineRatioBase = decimal.NewFromInt(10000)
	base           = decimal.NewFromInt(10000)
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
