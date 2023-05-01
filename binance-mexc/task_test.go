package binancemexc

import (
	"testing"

	"github.com/isther/arbitrage/config"
)

func TestArbitrageTask(t *testing.T) {
	config.Load("../config.yaml")
	symbolPair := SymbolPair{
		BinanceSymbol:    "BTCTUSD",
		StableCoinSymbol: "TUSDUSDT",
		MexcSymbol:       "BTCUSDT",
	}
	ArbitrageManagerInstance = NewArbitrageManager(symbolPair)

	go NewArbitrageTask(
		config.Config.BinanceApiKey,
		config.Config.BinanceSecretKey,
		config.Config.MexcApiKey,
		config.Config.MexcSecretKey,
		symbolPair,
		0.0007,
		0.0015,
	)

	ArbitrageManagerInstance.StartWsBookTicker()
}
