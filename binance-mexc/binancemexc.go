package binancemexc

import (
	"sync/atomic"

	"github.com/shopspring/decimal"
)

const (
	qty = "0.0004"
)

var (
	Paused         atomic.Bool
	pauseCh        = make(chan struct{}, 2)
	unPauseCh      = make(chan struct{}, 2)
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
