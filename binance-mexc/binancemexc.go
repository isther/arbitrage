package binancemexc

import (
	"sync/atomic"
	"time"

	"github.com/shopspring/decimal"
)

const (
	qty = "0.0004"
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
	go RunPause()
}

func RunPause() {
	for {
		select {
		case <-pauseCh:
			Paused.Store(true)
		case <-unPauseCh:
			Paused.Store(false)
		}
	}
}

func truelyPause() {
	for {
		switch Paused.Load() {
		case true:
			time.Sleep(10 * time.Millisecond)
		case false:
			pauseCh <- struct{}{}
			return
		}
	}
}
