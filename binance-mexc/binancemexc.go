package binancemexc

import (
	"sync/atomic"
	"time"

	"github.com/shopspring/decimal"
)

var (
	Paused                                atomic.Bool
	pauseCh                                     = make(chan struct{})
	unPauseCh                                   = make(chan struct{})
	klineRatioBase                              = decimal.NewFromInt(10000)
	base                                        = decimal.NewFromInt(10000)
	reconnectMexcAccountInfoSleepDuration int64 = 1000 //ms
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
			truelyPause()
		case false:
			pauseCh <- struct{}{}
			return
		}
	}
}
