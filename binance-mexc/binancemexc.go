package binancemexc

import (
	"sync/atomic"

	"github.com/shopspring/decimal"
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
