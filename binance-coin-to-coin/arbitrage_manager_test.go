package bimc

import (
	"testing"
	"time"
)

func TestArbitrageManagerStart(t *testing.T) {
	arbitrageManager := NewArbitrageManager("BTCTUSD", "TUSDUSDT", "BUSDUSDT")

	go arbitrageManager.StartWsBookTicker()

	arbitrageManager.addSymbol("BTCUSDT", "ETHUSDT", "TUSDUSDT").Restart()
	time.Sleep(1 * time.Second)
	arbitrageManager.addSymbol("BTCBUSD", "ETHBUSD").Restart()
	time.Sleep(5 * time.Second)
}
